package compiler

import (
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/vm"
)

type Compiler struct {
	chunk *vm.Chunk
}

func New() *Compiler {
	return &Compiler{
		chunk: &vm.Chunk{},
	}
}

func (c *Compiler) Compile(prog *parser.Program) *vm.Chunk {
	for _, stmt := range prog.Statements {
		c.compileStmt(stmt)
	}
	c.emit(vm.OpReturn)
	return c.chunk
}

func (c *Compiler) compileStmt(stmt *parser.Statement) {
	if stmt.Expr != nil {
		c.compileExpr(stmt.Expr)
		// Only print if it's not a print call
		if stmt.Expr.Call == nil || stmt.Expr.Call.Name != "print" {
			c.emit(vm.OpPrint)
		}
	} else if stmt.Let != nil {
		c.compileExpr(stmt.Let.Expr)
		// Store the value in globals
		idx := c.addConstant(stmt.Let.Name)
		c.emit(vm.OpDefineGlobal, byte(idx))
	}
}

func (c *Compiler) compileExpr(expr *parser.Expr) {
	if expr == nil {
		return
	}
	switch {
	case expr.Term != nil:
		t := expr.Term
		if t.Number != nil {
			idx := c.addConstant(*t.Number)
			c.emit(vm.OpConstant, byte(idx))
		}
		if t.String != nil {
			idx := c.addConstant(*t.String)
			c.emit(vm.OpConstant, byte(idx))
		}
		if t.Bool != nil {
			idx := c.addConstant(*t.Bool)
			c.emit(vm.OpConstant, byte(idx))
		}
		if t.Ident != nil {
			idx := c.addConstant(*t.Ident)
			c.emit(vm.OpGetGlobal, byte(idx))
		}
	case expr.Block != nil:
		// Compile each expression in the block
		for _, e := range expr.Block.Exprs {
			c.compileExpr(e)
		}
	case expr.If != nil:
		// Compile condition
		c.compileExpr(expr.If.Cond)
		// Emit jump if false
		jumpIfFalsePos := len(c.chunk.Code)
		c.emit(vm.OpJumpIfFalse, 0) // placeholder
		// Compile then block
		c.compileExpr(expr.If.ThenExpr)
		// Emit jump to end
		jumpToEndPos := len(c.chunk.Code)
		c.emit(vm.OpJump, 0) // placeholder
		// Update jump if false offset
		elsePos := len(c.chunk.Code)
		c.chunk.Code[jumpIfFalsePos+1] = byte(elsePos)
		// Compile else block
		c.compileExpr(expr.If.ElseExpr)
		// Update jump to end offset
		endPos := len(c.chunk.Code)
		c.chunk.Code[jumpToEndPos+1] = byte(endPos)
	case expr.Func != nil:
		// Create a new chunk for the function body
		fnChunk := &vm.Chunk{
			Params: expr.Func.Params,
		}
		// Save current chunk
		oldChunk := c.chunk
		// Set current chunk to function chunk
		c.chunk = fnChunk
		// Compile function body
		c.compileExpr(expr.Func.Body)
		c.emit(vm.OpReturn)
		// Restore original chunk
		c.chunk = oldChunk
		// Add function chunk to constants
		idx := c.addConstant(fnChunk)
		c.emit(vm.OpClosure, byte(idx))
	case expr.Call != nil:
		// Push function name
		idx := c.addConstant(expr.Call.Name)
		c.emit(vm.OpConstant, byte(idx))
		// Push arguments
		for _, arg := range expr.Call.Args {
			c.compileExpr(arg)
		}
		// Call function with number of arguments
		c.emit(vm.OpCall, byte(len(expr.Call.Args)))
	case expr.Bin != nil:
		if expr.Bin.Left != nil {
			c.compileExpr(&parser.Expr{Term: expr.Bin.Left})
		}
		if expr.Bin.Right != nil {
			c.compileExpr(expr.Bin.Right)
		}
		if expr.Bin.Operator != nil {
			switch *expr.Bin.Operator {
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
			}
		}
	}
}

func (c *Compiler) emit(op vm.Opcode, operands ...byte) {
	c.chunk.Code = append(c.chunk.Code, byte(op))
	c.chunk.Code = append(c.chunk.Code, operands...)
}

func (c *Compiler) addConstant(val interface{}) int {
	c.chunk.Constants = append(c.chunk.Constants, val)
	return len(c.chunk.Constants) - 1
}
