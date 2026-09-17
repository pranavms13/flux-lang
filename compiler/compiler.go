package compiler

import (
	"encoding/binary"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/vm"
)

type FluxCompiler struct {
	chunk *vm.Chunk
	// src resolves node positions into locations that survive serialization.
	// It is nil for a compiler built without a source, and the chunks then
	// carry no source map.
	src *source.Source
	// name is the identifier the next function literal is being bound to, so
	// that a call trace can say "add" rather than "<anonymous>".
	name string
}

func NewFluxCompiler() *FluxCompiler { return &FluxCompiler{} }

// NewFluxCompilerForSource returns a compiler that records where each
// instruction came from. The program it is given must have been parsed from src.
func NewFluxCompilerForSource(src *source.Source) *FluxCompiler {
	return &FluxCompiler{src: src}
}

func (c *FluxCompiler) Compile(prog *ast.Program) *vm.Chunk {
	c.chunk = &vm.Chunk{}
	for _, stmt := range prog.Statements {
		if stmt.Let != nil {
			c.name = stmt.Let.Name
			c.compileExpr(stmt.Let.Expr)
			c.name = ""
			c.emitAt(stmt.Let, vm.OpDefineGlobal, c.addConstant(stmt.Let.Name))
		} else {
			c.compileExpr(stmt.Expr)
			c.emitAt(stmt.Expr, vm.OpPrint)
		}
	}
	c.emitAt(prog, vm.OpReturn)
	return c.chunk
}

// Every expression leaves exactly one value on the stack (nil for void).
func (c *FluxCompiler) compileExpr(expr *ast.Expr) {
	switch {
	case expr.Primary != nil:
		c.compilePrimary(expr.Primary)
	case expr.Block != nil:
		c.compileBlock(expr.Block)
	case expr.If != nil:
		c.compileExpr(expr.If.Cond)
		falseJump := c.emitAt(expr.If.Cond, vm.OpJumpIfFalse, 0)
		c.compileExpr(expr.If.ThenExpr)
		endJump := c.emitAt(expr.If.ThenExpr, vm.OpJump, 0)
		c.patchJump(falseJump)
		c.compileExpr(expr.If.ElseExpr)
		c.patchJump(endJump)
	case expr.Func != nil:
		params := make([]string, len(expr.Func.Params))
		for i, p := range expr.Func.Params {
			params[i] = p.Name
		}
		outer, outerName := c.chunk, c.name
		// A nested function gets its own source map, so a failure inside it
		// reports its own line rather than the line of the call.
		c.chunk = &vm.Chunk{Params: params, Name: outerName}
		c.name = ""
		c.compileExpr(expr.Func.Body)
		c.emitAt(expr.Func.Body, vm.OpReturn)
		fn := c.chunk
		c.chunk, c.name = outer, outerName
		c.emitAt(expr.Func, vm.OpClosure, c.addConstant(fn))
	case expr.Bin != nil:
		c.compileAdditive(expr.Bin.Left)
		for _, rest := range expr.Bin.Rest {
			c.compileAdditive(rest.Right)
			c.compileOperator(rest, rest.Operator)
		}
	}
}

func (c *FluxCompiler) compileAdditive(expr *ast.Additive) {
	c.compilePrimary(expr.Left)
	for _, rest := range expr.Rest {
		c.compilePrimary(rest.Right)
		c.compileOperator(rest, rest.Operator)
	}
}

func (c *FluxCompiler) compileOperator(at ast.Positioned, op string) {
	switch op {
	case "+":
		c.emitAt(at, vm.OpAdd)
	case "-":
		c.emitAt(at, vm.OpSub)
	case "==":
		c.emitAt(at, vm.OpEqual)
	case ">":
		c.emitAt(at, vm.OpGreater)
	case "<":
		c.emitAt(at, vm.OpLess)
	default:
		panic("unsupported operator: " + op)
	}
}

func (c *FluxCompiler) compileBlock(block *ast.BlockExpr) {
	if len(block.Exprs) == 0 {
		c.emitAt(block, vm.OpConstant, c.addConstant(nil))
		return
	}
	for i, expr := range block.Exprs {
		c.compileExpr(expr)
		if i < len(block.Exprs)-1 {
			c.emitAt(expr, vm.OpPop)
		}
	}
}

func (c *FluxCompiler) compilePrimary(expr *ast.PrimaryExpr) {
	base := expr.Base
	switch {
	case base.Term != nil:
		t := base.Term
		switch {
		case t.Number != nil:
			c.emitAt(t, vm.OpConstant, c.addConstant(*t.Number))
		case t.String != nil:
			c.emitAt(t, vm.OpConstant, c.addConstant(*t.String))
		case t.Bool != nil:
			c.emitAt(t, vm.OpConstant, c.addConstant(bool(*t.Bool)))
		case t.Ident != nil:
			c.emitAt(t, vm.OpGetGlobal, c.addConstant(*t.Ident))
		}
	case base.Group != nil:
		c.compileExpr(base.Group.Expr)
	case base.Block != nil:
		c.compileBlock(base.Block)
	case base.List != nil:
		for _, elem := range base.List.Elems {
			c.compileExpr(elem)
		}
		c.emitAt(base.List, vm.OpArray, len(base.List.Elems))
	case base.Dict != nil:
		for _, pair := range base.Dict.Pairs {
			c.compileExpr(pair.Key)
			c.compileExpr(pair.Value)
		}
		c.emitAt(base.Dict, vm.OpDict, len(base.Dict.Pairs))
	}
	for _, postfix := range expr.Postfix {
		if postfix.Call != nil {
			for _, arg := range postfix.Call.Args {
				c.compileExpr(arg)
			}
			c.emitAt(postfix.Call, vm.OpCall, len(postfix.Call.Args))
		} else {
			c.compileExpr(postfix.Index.Index)
			c.emitAt(postfix.Index, vm.OpIndex)
		}
	}
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
