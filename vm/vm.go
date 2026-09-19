package vm

import (
	"encoding/binary"
	"fmt"
	"github.com/pranavms13/flux-lang/value"
	"io"
	"os"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/source"
)

type Opcode byte

const (
	OpConstant Opcode = iota
	OpAdd
	OpSub
	OpEqual
	OpGreater
	OpLess
	OpPop
	OpPrint
	OpReturn
	OpDefineGlobal
	OpGetGlobal
	OpCall
	OpClosure
	OpJumpIfFalse
	OpJump
	OpJumpIfTrue
	OpIndex
	OpArray
	OpDict
	OpMultiply
	OpDivide
	OpRemainder
	OpNotEqual
	OpLessEqual
	OpGreaterEqual
	OpNegate
	OpNot
	OpCheckBool
	OpCheckKey
	OpDeclareGlobal
	OpDeclareLocal
	OpDefineLocal
	OpGetLocal
	OpGetCapture
	OpEnterScope
	OpLeaveScope
)

// Operands returns how many four-byte operands an opcode carries.
//
// It is what makes a chunk walkable: instructions are variable length, so
// anything that steps through code — a disassembler, or a test checking that
// the source map is keyed at instruction boundaries — needs this table rather
// than a guess.
func (o Opcode) Operands() int {
	switch o {
	case OpConstant, OpDefineGlobal, OpGetGlobal, OpCall, OpClosure,
		OpJumpIfFalse, OpJumpIfTrue, OpJump, OpArray, OpDict,
		OpDeclareGlobal, OpDeclareLocal, OpDefineLocal, OpGetLocal, OpGetCapture:
		return 1
	default:
		return 0
	}
}

type Chunk struct {
	Code         []byte
	Constants    []interface{}
	Params       []string
	ParamIDs     []uint32
	Captures     []uint32
	BindingCount uint32
	Failure      *fault.Error
	// Name labels the function in a call trace. It is the name the function
	// was bound to, or empty for a literal that was never named.
	Name string
	// Locations maps the byte offset of an instruction's opcode to where the
	// construct that produced it was written. It is keyed by the instruction
	// start, not by any later byte, so a failure reports the operation that
	// failed rather than the one that follows it.
	//
	// The locations are already resolved to file, line and column: a generated
	// executable carries this map inside itself and must still report a
	// position after the .flux file it was built from is gone.
	Locations map[int]source.Location
}

// Locate returns where the instruction starting at offset came from.
func (c *Chunk) Locate(offset int) source.Location { return c.Locations[offset] }

// Label returns the chunk's name for a call trace.
func (c *Chunk) Label() string {
	if c.Name == "" {
		return fault.Anonymous
	}
	return c.Name
}

type Closure struct {
	Chunk    *Chunk
	Captures map[uint32]*value.Cell
}

func (*Closure) FluxTypeName() string { return "function" }

type builtinPrint struct{}

func (builtinPrint) FluxTypeName() string { return "function" }

type VM struct {
	chunk    *Chunk
	ip       int
	stack    []interface{}
	globals  map[uint32]*value.Cell
	locals   map[uint32]*value.Cell
	captures map[uint32]*value.Cell
	scopes   []map[uint32]*value.Cell
	depth    int
	// MaxDepth limits active user function calls. Zero uses 256.
	MaxDepth int
	// depthLimit is MaxDepth with its default resolved. Run reads MaxDepth
	// rather than writing to it, so that a caller which configures a VM and
	// then inspects it never finds a value it did not assign.
	depthLimit int
	// boundaries holds the instruction start offsets of the chunk being run.
	boundaries map[int]bool
	// boundaryCache memoizes those offsets per chunk across a whole call
	// tree. The set is derived from Chunk.Code, which never changes after
	// compilation, but Run is entered once per user function call, so
	// rebuilding it there made every call pay for the whole chunk again. It
	// lives on the VM rather than on the Chunk so that two VMs may still run
	// one chunk concurrently.
	boundaryCache map[*Chunk]map[int]bool
	out           io.Writer
}

// New returns a VM that prints to standard output.
func New(chunk *Chunk) *VM { return NewWithOutput(chunk, os.Stdout) }

// NewWithOutput returns a VM that prints to out.
//
// Output is injected rather than taken from os.Stdout so that a test can read
// what a program printed without replacing a global, which is what forced the
// execution tests to run one at a time.
func NewWithOutput(chunk *Chunk, out io.Writer) *VM {
	if out == nil {
		out = os.Stdout
	}
	return &VM{
		chunk:   chunk,
		ip:      0,
		stack:   []interface{}{},
		globals: map[uint32]*value.Cell{1: {Value: builtinPrint{}, Initialized: true}},
		locals:  map[uint32]*value.Cell{},
		out:     out,
	}
}

// Run executes the chunk, returning the first failure it hits.
//
// Failures are returned rather than raised. A panic would be indistinguishable
// from a defect in the VM itself, and the CLI would have to guess which it had
// caught.
func (vm *VM) Run() error {
	if vm.chunk.Failure != nil {
		return vm.chunk.Failure
	}
	vm.depthLimit = vm.MaxDepth
	if vm.depthLimit <= 0 {
		vm.depthLimit = value.DefaultMaxDepth
	}
	vm.boundaries = vm.instructionBoundaries()

	for vm.ip < len(vm.chunk.Code) {
		// The instruction's own offset, captured before its operands are read,
		// so that a failure points at this operation and not the next one.
		start := vm.ip
		op := Opcode(vm.readByte())
		if err := vm.step(op, start); err != nil {
			return err
		}
		if op == OpReturn {
			return nil
		}
	}
	return nil
}

// instructionBoundaries returns the offsets at which the current chunk's
// instructions start, which are the only offsets a jump may target. The end of
// the code counts as a boundary so that a jump past the last instruction can
// terminate the loop.
//
// The answer is cached per chunk: a recursive function enters Run once per
// call, and walking its whole body to rediscover a fixed property of the
// bytecode dominated the cost of running it.
func (vm *VM) instructionBoundaries() map[int]bool {
	if vm.boundaryCache == nil {
		vm.boundaryCache = map[*Chunk]map[int]bool{}
	} else if cached, ok := vm.boundaryCache[vm.chunk]; ok {
		return cached
	}
	boundaries := map[int]bool{len(vm.chunk.Code): true}
	for ip := 0; ip < len(vm.chunk.Code); {
		boundaries[ip] = true
		ip += 1 + 4*Opcode(vm.chunk.Code[ip]).Operands()
	}
	vm.boundaryCache[vm.chunk] = boundaries
	return boundaries
}

// step validates operand availability and executes one instruction, reporting
// failures at the instruction start.
func (vm *VM) step(op Opcode, start int) error {
	// Validate the entire operand before any opcode reads it or changes state.
	if len(vm.chunk.Code)-vm.ip < 4*op.Operands() {
		return vm.internal(start, "instruction at %d is missing its operand", start)
	}
	for offset := 0; offset < op.Operands(); offset++ {
		if uint64(binary.BigEndian.Uint32(vm.chunk.Code[vm.ip+offset*4:vm.ip+offset*4+4])) > uint64(^uint(0)>>1) {
			return vm.internal(start, "operand exceeds host index range")
		}
	}
	switch op {
	case OpConstant:
		index := vm.readOperand()
		if index >= len(vm.chunk.Constants) {
			return vm.internal(start, "constant %d does not exist", index)
		}
		vm.push(vm.chunk.Constants[index])
	case OpIndex:
		if err := vm.need(2, start, op); err != nil {
			return err
		}
		index := vm.pop()
		value := vm.pop()
		result, err := valueIndex(value, index)
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		vm.push(result)
	case OpDict:
		size := vm.readOperand()
		if size > len(vm.stack)/2 {
			return vm.internal(start, "dictionary needs more stack values")
		}
		if err := vm.need(size*2, start, op); err != nil {
			return err
		}
		keys := make([]interface{}, size)
		values := make([]interface{}, size)
		for i := size - 1; i >= 0; i-- {
			values[i] = vm.pop()
			keys[i] = vm.pop()
		}
		dict := make(map[interface{}]interface{}, size)
		for i, key := range keys {
			if err := value.Key(key); err != nil {
				return vm.locatedFailure(err, start)
			}
			dict[key] = values[i]
		}
		vm.push(dict)
	case OpArray:
		size := vm.readOperand()
		if err := vm.need(size, start, op); err != nil {
			return err
		}
		elems := make([]interface{}, size)
		for i := size - 1; i >= 0; i-- {
			elems[i] = vm.pop()
		}
		vm.push(elems)
	case OpAdd, OpSub, OpGreater, OpLess, OpMultiply, OpDivide, OpRemainder, OpLessEqual, OpGreaterEqual, OpEqual, OpNotEqual:
		if err := vm.need(2, start, op); err != nil {
			return err
		}
		b, a := vm.pop(), vm.pop()
		result, err := value.Binary(operatorName(op), a, b)
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		vm.push(result)
	case OpNegate, OpNot:
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		name := "-"
		if op == OpNot {
			name = "!"
		}
		result, err := value.Unary(name, vm.pop())
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		vm.push(result)
	case OpCheckBool:
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		if _, err := value.Bool(vm.stack[len(vm.stack)-1]); err != nil {
			return vm.locatedFailure(err, start)
		}
	case OpCheckKey:
		if err := vm.need(2, start, op); err != nil {
			return err
		}
		if err := value.Key(vm.stack[len(vm.stack)-2]); err != nil {
			return vm.locatedFailure(err, start)
		}
	case OpPop:
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		vm.pop()
	case OpPrint:
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		if val := vm.pop(); val != nil {
			fmt.Fprintln(vm.out, value.Display(val))
		}
	case OpDeclareGlobal, OpDeclareLocal, OpDefineGlobal, OpDefineLocal, OpGetGlobal, OpGetLocal, OpGetCapture:
		id := uint32(vm.readOperand())
		if id == 0 || id > vm.chunk.BindingCount {
			return vm.internal(start, "binding %d is out of bounds", id)
		}
		env := vm.locals
		if op == OpDeclareGlobal || op == OpDefineGlobal || op == OpGetGlobal {
			env = vm.globals
		} else if op == OpGetCapture {
			env = vm.captures
		}
		switch op {
		case OpDeclareGlobal, OpDeclareLocal:
			if _, ok := env[id]; ok {
				return vm.internal(start, "binding %d is already allocated", id)
			}
			env[id] = &value.Cell{}
		case OpDefineGlobal, OpDefineLocal:
			if err := vm.need(1, start, op); err != nil {
				return err
			}
			cell := env[id]
			if cell == nil || cell.Initialized {
				return vm.internal(start, "binding %d cannot be initialized", id)
			}
			cell.Value, cell.Initialized = vm.pop(), true
		default:
			cell := env[id]
			if cell == nil || !cell.Initialized {
				return vm.internal(start, "binding %d is unavailable", id)
			}
			vm.push(cell.Value)
		}
	case OpEnterScope:
		vm.scopes = append(vm.scopes, vm.locals)
		vm.locals = copyCells(vm.locals)
	case OpLeaveScope:
		if len(vm.scopes) == 0 {
			return vm.internal(start, "scope stack is empty")
		}
		vm.locals = vm.scopes[len(vm.scopes)-1]
		vm.scopes = vm.scopes[:len(vm.scopes)-1]
	case OpJumpIfFalse, OpJumpIfTrue:
		offset := vm.readOperand()
		if !vm.boundaries[offset] {
			return vm.internal(start, "invalid jump target %d", offset)
		}
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		b, err := value.Bool(vm.pop())
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		if b == (op == OpJumpIfTrue) {
			vm.ip = offset
		}
	case OpJump:
		offset := vm.readOperand()
		if !vm.boundaries[offset] {
			return vm.internal(start, "invalid jump target %d", offset)
		}
		vm.ip = offset
	case OpCall:
		return vm.call(vm.readOperand(), start)
	case OpClosure:
		index := vm.readOperand()
		if index >= len(vm.chunk.Constants) {
			return vm.internal(start, "constant %d does not exist", index)
		}
		fnChunk, ok := vm.chunk.Constants[index].(*Chunk)
		if !ok {
			return vm.internal(start, "constant %d is not a function", index)
		}
		captures := map[uint32]*value.Cell{}
		for _, id := range fnChunk.Captures {
			if id == 0 || id > vm.chunk.BindingCount {
				return vm.internal(start, "capture %d is out of bounds", id)
			}
			cell := vm.locals[id]
			if cell == nil {
				cell = vm.captures[id]
			}
			if cell == nil {
				return vm.internal(start, "capture %d is unavailable", id)
			}
			captures[id] = cell
		}
		vm.push(&Closure{Chunk: fnChunk, Captures: captures})
	case OpReturn:
		return nil
	default:
		return vm.internal(start, "unknown opcode %d", op)
	}
	return nil
}

// call consumes a callee and its arguments, runs a closure or builtin, and
// pushes the result while preserving call traces on failure.
func (vm *VM) call(nargs, start int) error {
	if nargs >= len(vm.stack) {
		return vm.internal(start, "call needs more stack values")
	}
	if err := vm.need(nargs+1, start, OpCall); err != nil {
		return err
	}
	args := make([]interface{}, nargs)
	for i := nargs - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}
	callee := vm.pop()

	switch fn := callee.(type) {
	case *Closure:
		if len(args) != len(fn.Chunk.Params) {
			return vm.locatedFailure(
				fault.ArgumentCount(fn.Chunk.Label(), len(fn.Chunk.Params), len(args)), start)
		}
		if vm.depth >= vm.depthLimit {
			return vm.locatedFailure(fault.CallDepth(vm.depthLimit), start)
		}
		if len(fn.Chunk.ParamIDs) != len(args) {
			return vm.internal(start, "function parameter metadata is invalid")
		}
		subVM := NewWithOutput(fn.Chunk, vm.out)
		subVM.globals = vm.globals
		subVM.captures = fn.Captures
		subVM.depth, subVM.MaxDepth = vm.depth+1, vm.depthLimit
		subVM.boundaryCache = vm.boundaryCache
		for i, id := range fn.Chunk.ParamIDs {
			if id == 0 || id > fn.Chunk.BindingCount {
				return vm.internal(start, "parameter %d is out of bounds", id)
			}
			if subVM.locals[id] != nil {
				return vm.internal(start, "duplicate parameter slot %d", id)
			}
			subVM.locals[id] = &value.Cell{Value: args[i], Initialized: true}
		}
		if err := subVM.Run(); err != nil {
			// The failure unwinds through this call, so the frame naming the
			// function and the call site is added here, innermost first.
			return addFrame(err, fn.Chunk.Label(), vm.chunk.Locate(start))
		}
		if len(subVM.stack) != 1 {
			return vm.internal(start, "function returned %d values, expected one", len(subVM.stack))
		}
		vm.push(subVM.pop())
	case builtinPrint:
		if len(args) != 1 {
			return vm.locatedFailure(fault.ArgumentCount("print", 1, len(args)), start)
		}
		fmt.Fprintln(vm.out, value.Display(args[0]))
		vm.push(nil)
	default:
		return vm.locatedFailure(fault.NotCallable(callee), start)
	}
	return nil
}

func valueIndex(v, index any) (any, error) { return value.Index(v, index) }
func copyCells(env map[uint32]*value.Cell) map[uint32]*value.Cell {
	result := make(map[uint32]*value.Cell, len(env))
	for id, cell := range env {
		result[id] = cell
	}
	return result
}

// operatorName is the source spelling of a binary opcode, used to name the
// operation in a diagnostic. It is a switch rather than a map literal because
// it runs once per arithmetic and comparison instruction, where building a map
// would allocate on every operation.
//
// An opcode outside the binary set can only come from a hand-built or damaged
// chunk. It is named rather than rendered as the empty string so that the
// resulting diagnostic still says which instruction it came from.
func operatorName(op Opcode) string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMultiply:
		return "*"
	case OpDivide:
		return "/"
	case OpRemainder:
		return "%"
	case OpEqual:
		return "=="
	case OpNotEqual:
		return "!="
	case OpLess:
		return "<"
	case OpLessEqual:
		return "<="
	case OpGreater:
		return ">"
	case OpGreaterEqual:
		return ">="
	default:
		return fmt.Sprintf("opcode %d", op)
	}
}

// need verifies that an instruction has the operands it is about to consume.
// A shortfall is a defect in the compiler, not a mistake in the program, so it
// is reported as one.
func (vm *VM) need(n, start int, op Opcode) error {
	if len(vm.stack) < n {
		return vm.internal(start, "opcode %d needs %d stack %s, found %d",
			op, n, plural(n, "value"), len(vm.stack))
	}
	return nil
}

// locatedFailure attaches the position of the failing instruction to a
// catalogue entry that was built without one.
func (vm *VM) locatedFailure(err error, start int) error {
	if failure, ok := err.(*fault.Error); ok {
		return failure.At(vm.chunk.Locate(start))
	}
	return err
}

// internal reports a defect in Flux itself. It is deliberately a different
// code from anything in the runtime catalogue: a user program must not be able
// to provoke one, and the CLI must not present one as a mistake in the program.
func (vm *VM) internal(start int, format string, args ...interface{}) error {
	return &fault.Error{
		Code:    diagnostic.CodeInternal,
		Message: fmt.Sprintf(format, args...),
		Where:   vm.chunk.Locate(start),
	}
}

// addFrame appends a call site to a runtime fault and leaves other errors
// unchanged.
func addFrame(err error, function string, call source.Location) error {
	if failure, ok := err.(*fault.Error); ok {
		return failure.WithTrace([]fault.Frame{{Function: function, Call: call}})
	}
	return err
}

// plural uses the singular noun only when the count is one.
func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}

func (vm *VM) push(val interface{}) {
	vm.stack = append(vm.stack, val)
}

func (vm *VM) pop() interface{} {
	if len(vm.stack) == 0 {
		return nil
	}
	val := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return val
}

func (vm *VM) readByte() byte {
	b := vm.chunk.Code[vm.ip]
	vm.ip++
	return b
}

func (vm *VM) readOperand() int {
	operand := binary.BigEndian.Uint32(vm.chunk.Code[vm.ip : vm.ip+4])
	vm.ip += 4
	return int(operand)
}
