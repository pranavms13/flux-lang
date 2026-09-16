package compiler

import (
	"encoding/binary"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/vm"
)

type FluxCompiler struct{ chunk *vm.Chunk }

func NewFluxCompiler() *FluxCompiler { return &FluxCompiler{} }

func (c *FluxCompiler) Compile(prog *ast.Program) *vm.Chunk {
	c.chunk = &vm.Chunk{}
	for _, stmt := range prog.Statements {
		if stmt.Let != nil {
			c.compileExpr(stmt.Let.Expr)
			c.emit(vm.OpDefineGlobal, c.addConstant(stmt.Let.Name))
		} else {
			c.compileExpr(stmt.Expr)
			c.emit(vm.OpPrint)
		}
	}
	c.emit(vm.OpReturn)
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
		falseJump := c.emit(vm.OpJumpIfFalse, 0)
		c.compileExpr(expr.If.ThenExpr)
		endJump := c.emit(vm.OpJump, 0)
		c.patchJump(falseJump)
		c.compileExpr(expr.If.ElseExpr)
		c.patchJump(endJump)
	case expr.Func != nil:
		params := make([]string, len(expr.Func.Params))
		for i, p := range expr.Func.Params {
			params[i] = p.Name
		}
		outer := c.chunk
		c.chunk = &vm.Chunk{Params: params}
		c.compileExpr(expr.Func.Body)
		c.emit(vm.OpReturn)
		fn := c.chunk
		c.chunk = outer
		c.emit(vm.OpClosure, c.addConstant(fn))
	case expr.Bin != nil:
		c.compileAdditive(expr.Bin.Left)
		for _, rest := range expr.Bin.Rest {
			c.compileAdditive(rest.Right)
			c.compileOperator(rest.Operator)
		}
	}
}

func (c *FluxCompiler) compileAdditive(expr *ast.Additive) {
	c.compilePrimary(expr.Left)
	for _, rest := range expr.Rest {
		c.compilePrimary(rest.Right)
		c.compileOperator(rest.Operator)
	}
}

func (c *FluxCompiler) compileOperator(op string) {
	switch op {
	case "+":
		c.emit(vm.OpAdd)
	case "-":
		c.emit(vm.OpSub)
	case "==":
		c.emit(vm.OpEqual)
	case ">":
		c.emit(vm.OpGreater)
	case "<":
		c.emit(vm.OpLess)
	default:
		panic("unsupported operator: " + op)
	}
}

func (c *FluxCompiler) compileBlock(block *ast.BlockExpr) {
	if len(block.Exprs) == 0 {
		c.emit(vm.OpConstant, c.addConstant(nil))
		return
	}
	for i, expr := range block.Exprs {
		c.compileExpr(expr)
		if i < len(block.Exprs)-1 {
			c.emit(vm.OpPop)
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
			c.emit(vm.OpConstant, c.addConstant(*t.Number))
		case t.String != nil:
			c.emit(vm.OpConstant, c.addConstant(*t.String))
		case t.Bool != nil:
			c.emit(vm.OpConstant, c.addConstant(bool(*t.Bool)))
		case t.Ident != nil:
			c.emit(vm.OpGetGlobal, c.addConstant(*t.Ident))
		}
	case base.Group != nil:
		c.compileExpr(base.Group.Expr)
	case base.Block != nil:
		c.compileBlock(base.Block)
	case base.List != nil:
		for _, elem := range base.List.Elems {
			c.compileExpr(elem)
		}
		c.emit(vm.OpArray, len(base.List.Elems))
	case base.Dict != nil:
		for _, pair := range base.Dict.Pairs {
			c.compileExpr(pair.Key)
			c.compileExpr(pair.Value)
		}
		c.emit(vm.OpDict, len(base.Dict.Pairs))
	}
	for _, postfix := range expr.Postfix {
		if postfix.Call != nil {
			for _, arg := range postfix.Call.Args {
				c.compileExpr(arg)
			}
			c.emit(vm.OpCall, len(postfix.Call.Args))
		} else {
			c.compileExpr(postfix.Index.Index)
			c.emit(vm.OpIndex)
		}
	}
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
