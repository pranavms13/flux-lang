package compiler

import (
	"encoding/binary"
	"math"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/resolver"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/vm"
)

type FluxCompiler struct {
	chunk    *vm.Chunk
	src      *source.Source
	name     string
	bindings *resolver.Result
	function *ast.FuncExpr
}

func NewFluxCompiler() *FluxCompiler                            { return &FluxCompiler{} }
func NewFluxCompilerForSource(src *source.Source) *FluxCompiler { return &FluxCompiler{src: src} }

// Compile resolves lexical bindings and emits format-2 bytecode. Invalid
// bindings produce a chunk carrying a structured failure, so direct API users
// cannot bypass resolution by omitting optional type checking. CLI callers
// receive the resolver diagnostics through CheckProgram before compiling.
func (c *FluxCompiler) Compile(prog *ast.Program) *vm.Chunk {
	c.bindings = resolver.Resolve(prog, c.src)
	c.chunk = &vm.Chunk{BindingCount: uint32(len(c.bindings.Bindings)), Failure: c.bindings.Failure(c.src)}
	c.function = nil
	c.name = ""
	if c.chunk.Failure != nil {
		return c.chunk
	}
	for _, s := range prog.Statements {
		c.statement(s)
		if s.Expr != nil {
			c.emitAt(s.Expr, vm.OpPrint)
		} else {
			c.emitAt(s.Let, vm.OpPop)
		}
	}
	c.emitAt(prog, vm.OpReturn)
	return c.chunk
}

// Statements and expressions each leave one value; declarations leave void.
func (c *FluxCompiler) statement(s *ast.Statement) {
	if s.Let == nil {
		c.compile(s.Expr)
		return
	}
	id := c.bindings.Declarations[s.Let]
	declare, define := vm.OpDeclareLocal, vm.OpDefineLocal
	if c.bindings.Bindings[id].Global {
		declare, define = vm.OpDeclareGlobal, vm.OpDefineGlobal
	}
	c.emitAt(s.Let, declare, int(id))
	oldName := c.name
	c.name = s.Let.Name
	c.compile(s.Let.Expr)
	c.name = oldName
	c.emitAt(s.Let, define, int(id))
	c.emitAt(s.Let, vm.OpConstant, c.addConstant(nil))
}
func (c *FluxCompiler) compile(n ast.Positioned) {
	if left, rest := ast.Chain(n); left != nil {
		c.compile(left)
		var at ast.Positioned = left
		for _, op := range rest {
			if op.Operator == "&&" || op.Operator == "||" {
				jump := vm.OpJumpIfFalse
				skip := false
				if op.Operator == "||" {
					jump = vm.OpJumpIfTrue
					skip = true
				}
				skipped := c.emitAt(at, jump, 0) // consumes and validates the left bool
				c.compile(op.Right)
				c.emitAt(op.Right, vm.OpCheckBool)
				end := c.emitAt(op.At, vm.OpJump, 0)
				c.patchJump(skipped)
				c.emitAt(op.At, vm.OpConstant, c.addConstant(skip))
				c.patchJump(end)
			} else {
				c.compile(op.Right)
				var location ast.Positioned = op.At
				if op.Operator == "==" || op.Operator == "!=" {
					location = ast.Range{Left: at, Right: op.At}
				}
				opcode, ok := operators[op.Operator]
				if !ok {
					panic("unsupported operator: " + op.Operator)
				}
				c.emitAt(location, opcode)
			}
			at = ast.Range{Left: at, Right: op.At}
		}
		return
	}
	switch n := n.(type) {
	case *ast.Expr:
		if n.If != nil {
			c.compile(n.If)
		} else if n.Func != nil {
			c.compile(n.Func)
		} else {
			c.compile(n.Bin)
		}
	case *ast.Unary:
		if n.MinLiteral {
			c.emitAt(n, vm.OpConstant, c.addConstant(int64(math.MinInt64)))
		} else if n.Primary != nil {
			c.compile(n.Primary)
		} else {
			c.compile(n.Operand)
			op := vm.OpNegate
			if n.Operator == "!" {
				op = vm.OpNot
			}
			c.emitAt(n, op)
		}
	case *ast.IfExpr:
		c.compile(n.Cond)
		otherwise := c.emitAt(n.Cond, vm.OpJumpIfFalse, 0)
		c.compile(n.ThenExpr)
		end := c.emitAt(n.ThenExpr, vm.OpJump, 0)
		c.patchJump(otherwise)
		c.compile(n.ElseExpr)
		c.patchJump(end)
	case *ast.FuncExpr:
		outer, outerName, outerFn := c.chunk, c.name, c.function
		c.chunk = &vm.Chunk{Name: c.name, BindingCount: outer.BindingCount}
		c.name = ""
		c.function = n
		for _, p := range n.Params {
			c.chunk.Params = append(c.chunk.Params, p.Name)
			c.chunk.ParamIDs = append(c.chunk.ParamIDs, uint32(c.bindings.Parameters[p]))
		}
		for _, id := range c.bindings.Functions[n].Captures {
			c.chunk.Captures = append(c.chunk.Captures, uint32(id))
		}
		c.compile(n.Body)
		c.emitAt(n.Body, vm.OpReturn)
		fn := c.chunk
		c.chunk, c.name, c.function = outer, outerName, outerFn
		c.emitAt(n, vm.OpClosure, c.addConstant(fn))
	case *ast.BlockExpr:
		c.emitAt(n, vm.OpEnterScope)
		if len(n.Statements) == 0 {
			c.emitAt(n, vm.OpConstant, c.addConstant(nil))
		}
		for i, s := range n.Statements {
			c.statement(s)
			if i < len(n.Statements)-1 {
				c.emitAt(s, vm.OpPop)
			}
		}
		c.emitAt(n, vm.OpLeaveScope)
	case *ast.PrimaryExpr:
		c.compile(n.Base)
		for _, p := range n.Postfix {
			if p.Call != nil {
				for _, a := range p.Call.Args {
					c.compile(a)
				}
				c.emitAt(p.Call, vm.OpCall, len(p.Call.Args))
			} else {
				c.compile(p.Index.Index)
				c.emitAt(p.Index, vm.OpIndex)
			}
		}
	case *ast.BaseExpr:
		switch {
		case n.Term != nil:
			c.compile(n.Term)
		case n.Group != nil:
			c.compile(n.Group.Expr)
		case n.Block != nil:
			c.compile(n.Block)
		case n.List != nil:
			c.compile(n.List)
		case n.Dict != nil:
			c.compile(n.Dict)
		}
	case *ast.Term:
		switch {
		case n.Number != nil:
			c.emitAt(n, vm.OpConstant, c.addConstant(n.Number.Value))
		case n.String != nil:
			c.emitAt(n, vm.OpConstant, c.addConstant(*n.String))
		case n.Bool != nil:
			c.emitAt(n, vm.OpConstant, c.addConstant(bool(*n.Bool)))
		case n.Ident != nil:
			id := c.bindings.Uses[n]
			b := c.bindings.Bindings[id]
			op := vm.OpGetLocal
			if b.Global {
				op = vm.OpGetGlobal
			} else if b.Function != c.function {
				op = vm.OpGetCapture
			}
			c.emitAt(n, op, int(id))
		}
	case *ast.ListExpr:
		for _, e := range n.Elems {
			c.compile(e)
		}
		c.emitAt(n, vm.OpArray, len(n.Elems))
	case *ast.DictExpr:
		for _, p := range n.Pairs {
			c.compile(p.Key)
			c.compile(p.Value)
			c.emitAt(p.Key, vm.OpCheckKey)
		}
		c.emitAt(n, vm.OpDict, len(n.Pairs))
	default:
		panic("unsupported AST node")
	}
}

var operators = map[string]vm.Opcode{
	"+": vm.OpAdd, "-": vm.OpSub, "*": vm.OpMultiply, "/": vm.OpDivide, "%": vm.OpRemainder,
	"==": vm.OpEqual, "!=": vm.OpNotEqual, "<": vm.OpLess, "<=": vm.OpLessEqual, ">": vm.OpGreater, ">=": vm.OpGreaterEqual,
}

// emitAt appends an instruction and records where it came from. The location
// is keyed by the instruction's own start offset, which is what the VM captures
// before it reads any operands.
func (c *FluxCompiler) emitAt(at ast.Positioned, op vm.Opcode, operands ...int) int {
	pos := c.emit(op, operands...)
	if c.src == nil || at == nil || !at.HasPosition() {
		return pos
	}
	if c.chunk.Locations == nil {
		c.chunk.Locations = map[int]source.Location{}
	}
	c.chunk.Locations[pos] = c.src.Locate(at.Span(c.src.ID()))
	return pos
}

func (c *FluxCompiler) emit(op vm.Opcode, operands ...int) int {
	pos := len(c.chunk.Code)
	c.chunk.Code = append(c.chunk.Code, byte(op))
	for _, operand := range operands {
		if operand < 0 || uint64(operand) > uint64(^uint32(0)) {
			panic("bytecode operand too large")
		}
		c.chunk.Code = binary.BigEndian.AppendUint32(c.chunk.Code, uint32(operand))
	}
	return pos
}

func (c *FluxCompiler) patchJump(pos int) {
	binary.BigEndian.PutUint32(c.chunk.Code[pos+1:pos+5], uint32(len(c.chunk.Code)))
}

func (c *FluxCompiler) addConstant(value interface{}) int {
	c.chunk.Constants = append(c.chunk.Constants, value)
	return len(c.chunk.Constants) - 1
}
