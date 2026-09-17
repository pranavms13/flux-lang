package vm

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"

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
		OpJumpIfFalse, OpJumpIfTrue, OpJump, OpArray, OpDict:
		return 1
	default:
		return 0
	}
}

type Chunk struct {
	Code      []byte
	Constants []interface{}
	Params    []string
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
	Chunk  *Chunk
	Locals map[string]interface{}
}

type builtinPrint struct{}

type VM struct {
	chunk   *Chunk
	ip      int
	stack   []interface{}
	globals map[string]interface{}
	locals  map[string]interface{}
	out     io.Writer
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
		globals: map[string]interface{}{},
		locals:  map[string]interface{}{},
		out:     out,
	}
}

// Run executes the chunk, returning the first failure it hits.
//
// Failures are returned rather than raised. A panic would be indistinguishable
// from a defect in the VM itself, and the CLI would have to guess which it had
// caught.
func (vm *VM) Run() error {
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

func (vm *VM) step(op Opcode, start int) error {
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
		result, err := indexValue(value, index)
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		vm.push(result)
	case OpDict:
		size := vm.readOperand()
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
	case OpAdd, OpSub, OpGreater, OpLess:
		if err := vm.need(2, start, op); err != nil {
			return err
		}
		b := vm.pop()
		a := vm.pop()
		result, err := arithmetic(op, a, b)
		if err != nil {
			return vm.locatedFailure(err, start)
		}
		vm.push(result)
	case OpEqual:
		if err := vm.need(2, start, op); err != nil {
			return err
		}
		b := vm.pop()
		a := vm.pop()
		vm.push(reflect.DeepEqual(a, b))
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
			fmt.Fprintln(vm.out, val)
		}
	case OpDefineGlobal:
		name, err := vm.constantName(start)
		if err != nil {
			return err
		}
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		vm.globals[name] = vm.pop()
	case OpGetGlobal:
		name, err := vm.constantName(start)
		if err != nil {
			return err
		}
		switch {
		case has(vm.locals, name):
			vm.push(vm.locals[name])
		case has(vm.globals, name):
			vm.push(vm.globals[name])
		case name == "print":
			vm.push(builtinPrint{})
		default:
			return vm.locatedFailure(fault.UndefinedValue(name), start)
		}
	case OpJumpIfFalse, OpJumpIfTrue:
		offset := vm.readOperand()
		if err := vm.need(1, start, op); err != nil {
			return err
		}
		if vm.truthy(vm.pop()) == (op == OpJumpIfTrue) {
			vm.ip = offset
		}
	case OpJump:
		vm.ip = vm.readOperand()
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
		// The closure keeps its chunk, which carries the function's name and
		// the source map its failures are reported through.
		locals := make(map[string]interface{}, len(vm.locals))
		for name, value := range vm.locals {
			locals[name] = value
		}
		vm.push(&Closure{Chunk: fnChunk, Locals: locals})
	case OpReturn:
		return nil
	default:
		return vm.internal(start, "unknown opcode %d", op)
	}
	return nil
}

func (vm *VM) call(nargs, start int) error {
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
		subVM := NewWithOutput(fn.Chunk, vm.out)
		subVM.globals = vm.globals
		for name, value := range fn.Locals {
			subVM.locals[name] = value
		}
		for i, param := range fn.Chunk.Params {
			subVM.locals[param] = args[i]
		}
		if err := subVM.Run(); err != nil {
			// The failure unwinds through this call, so the frame naming the
			// function and the call site is added here, innermost first.
			return addFrame(err, fn.Chunk.Label(), vm.chunk.Locate(start))
		}
		vm.push(subVM.pop())
	case builtinPrint:
		if len(args) != 1 {
			return vm.locatedFailure(fault.ArgumentCount("print", 1, len(args)), start)
		}
		fmt.Fprintln(vm.out, args[0])
		vm.push(nil)
	default:
		return vm.locatedFailure(fault.NotCallable(callee), start)
	}
	return nil
}

// indexValue is shared by every index failure so that the VM and the
// interpreter cannot drift apart on what "cannot index" means.
func indexValue(value, index interface{}) (interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		idx, ok := index.(int)
		if !ok {
			return nil, fault.IndexType(index)
		}
		if idx < 0 || idx >= len(v) {
			return nil, fault.IndexRange(idx, len(v))
		}
		return v[idx], nil
	case map[interface{}]interface{}:
		result, exists := v[index]
		if !exists {
			return nil, fault.MissingKey(index)
		}
		return result, nil
	default:
		return nil, fault.NotIndexable(value)
	}
}

func arithmetic(op Opcode, a, b interface{}) (interface{}, error) {
	switch op {
	case OpAdd:
		switch left := a.(type) {
		case int:
			if right, ok := b.(int); ok {
				return left + right, nil
			}
		case string:
			if right, ok := b.(string); ok {
				return left + right, nil
			}
		}
		return nil, fault.OperandType("+", a, b)
	case OpSub, OpGreater, OpLess:
		left, leftOK := a.(int)
		right, rightOK := b.(int)
		if !leftOK || !rightOK {
			return nil, fault.OperandType(operatorName(op), a, b)
		}
		switch op {
		case OpSub:
			return left - right, nil
		case OpGreater:
			return left > right, nil
		default:
			return left < right, nil
		}
	}
	return nil, fault.OperandType(operatorName(op), a, b)
}

func operatorName(op Opcode) string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpGreater:
		return ">"
	case OpLess:
		return "<"
	case OpEqual:
		return "=="
	default:
		return fmt.Sprintf("opcode %d", op)
	}
}

func has(bindings map[string]interface{}, name string) bool {
	_, ok := bindings[name]
	return ok
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

func (vm *VM) constantName(start int) (string, error) {
	index := vm.readOperand()
	if index >= len(vm.chunk.Constants) {
		return "", vm.internal(start, "constant %d does not exist", index)
	}
	name, ok := vm.chunk.Constants[index].(string)
	if !ok {
		return "", vm.internal(start, "constant %d is not a name", index)
	}
	return name, nil
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

func addFrame(err error, function string, call source.Location) error {
	if failure, ok := err.(*fault.Error); ok {
		return failure.WithTrace([]fault.Frame{{Function: function, Call: call}})
	}
	return err
}

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

func (vm *VM) truthy(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case string:
		return val != ""
	default:
		return val != nil
	}
}

func (vm *VM) readOperand() int {
	operand := binary.BigEndian.Uint32(vm.chunk.Code[vm.ip : vm.ip+4])
	vm.ip += 4
	return int(operand)
}
