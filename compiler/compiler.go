package compiler

import (
	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/vm"
)

type FluxCompiler struct {
	chunk *vm.Chunk
}

func NewFluxCompiler() *FluxCompiler {
	return &FluxCompiler{
		chunk: &vm.Chunk{},
	}
}

func (c *FluxCompiler) Compile(prog *ast.Program) *vm.Chunk {
	for _, stmt := range prog.Statements {
		c.compileStmt(stmt)
	}
	c.emit(vm.OpReturn)
	return c.chunk
}

func (c *FluxCompiler) compileStmt(stmt *ast.Statement) {
	if stmt.Expr != nil {
		c.compileExpr(stmt.Expr)
		// Only print if it's not a print call and not an array indexing
		isPrint := false
		isIndexing := false
		if stmt.Expr.Primary != nil {
			if stmt.Expr.Primary.Base != nil && stmt.Expr.Primary.Base.Term != nil && stmt.Expr.Primary.Base.Term.Ident != nil && *stmt.Expr.Primary.Base.Term.Ident == "print" {
				if len(stmt.Expr.Primary.Postfix) > 0 && stmt.Expr.Primary.Postfix[0].Call != nil {
					isPrint = true
				}
			}
			if len(stmt.Expr.Primary.Postfix) > 0 && stmt.Expr.Primary.Postfix[0].Index != nil {
				isIndexing = true
			}
		}
		if !isPrint && !isIndexing {
			c.emit(vm.OpPrint)
		}
	} else if stmt.Let != nil {
		c.compileExpr(stmt.Let.Expr)
		// Store the value in globals
		idx := c.addConstant(stmt.Let.Name)
		c.emit(vm.OpDefineGlobal, byte(idx))
	}
}

func (c *FluxCompiler) compileExpr(expr *ast.Expr) {
	if expr == nil {
		return
	}
	switch {
	case expr.Primary != nil:
		// Compile the base value
		if expr.Primary.Base != nil {
			if expr.Primary.Base.Term != nil {
				t := expr.Primary.Base.Term
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
			} else if expr.Primary.Base.List != nil {
				// First compile all elements
				for _, e := range expr.Primary.Base.List.Elems {
					c.compileExpr(e)
				}
				// Then create the array from the elements
				c.emit(vm.OpArray, byte(len(expr.Primary.Base.List.Elems)))
			} else if expr.Primary.Base.Dict != nil {
				// First compile all key-value pairs
				for _, pair := range expr.Primary.Base.Dict.Pairs {
					c.compileExpr(pair.Value)
					c.compileExpr(pair.Key)
				}
				// Then create the dictionary from the pairs
				c.emit(vm.OpDict, byte(len(expr.Primary.Base.Dict.Pairs)))
			}
		}
		// Compile chained postfix expressions
		for _, pf := range expr.Primary.Postfix {
			if pf.Call != nil {
				// First compile all arguments
				for _, arg := range pf.Call.Args {
					c.compileExpr(arg)
				}
				// Then emit the call instruction
				c.emit(vm.OpCall, byte(len(pf.Call.Args)))
			} else if pf.Index != nil {
				c.compileExpr(pf.Index.Index)
				// Check if we're accessing a dictionary by looking at the base expression
				if expr.Primary.Base != nil && expr.Primary.Base.Dict != nil {
					c.emit(vm.OpDictGet)
				} else if expr.Primary.Base != nil && expr.Primary.Base.List != nil {
					c.emit(vm.OpIndex)
				} else if expr.Primary.Base != nil && expr.Primary.Base.Term != nil && expr.Primary.Base.Term.Ident != nil {
					// For variable access, we need to check if it's a dictionary
					// We'll use OpIndex for now since we can't determine the type at compile time
					c.emit(vm.OpIndex)
				} else {
					c.emit(vm.OpIndex)
				}
			}
		}
	case expr.Block != nil:
		for _, e := range expr.Block.Exprs {
			c.compileExpr(e)
		}
	case expr.If != nil:
		c.compileExpr(expr.If.Cond)
		jumpIfFalsePos := len(c.chunk.Code)
		c.emit(vm.OpJumpIfFalse, 0)
		c.compileExpr(expr.If.ThenExpr)
		jumpToEndPos := len(c.chunk.Code)
		c.emit(vm.OpJump, 0)
		elsePos := len(c.chunk.Code)
		c.chunk.Code[jumpIfFalsePos+1] = byte(elsePos)
		c.compileExpr(expr.If.ElseExpr)
		endPos := len(c.chunk.Code)
		c.chunk.Code[jumpToEndPos+1] = byte(endPos)
	case expr.Func != nil:
		fnChunk := &vm.Chunk{
			Params: expr.Func.Params,
		}
		oldChunk := c.chunk
		c.chunk = fnChunk
		c.compileExpr(expr.Func.Body)
		c.emit(vm.OpReturn)
		c.chunk = oldChunk
		idx := c.addConstant(fnChunk)
		c.emit(vm.OpClosure, byte(idx))
	case expr.Bin != nil:
		if expr.Bin.Left != nil {
			c.compileExpr(&ast.Expr{Primary: expr.Bin.Left})
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

func (c *FluxCompiler) emit(op vm.Opcode, operands ...byte) {
	c.chunk.Code = append(c.chunk.Code, byte(op))
	c.chunk.Code = append(c.chunk.Code, operands...)
}

func (c *FluxCompiler) addConstant(val interface{}) int {
	c.chunk.Constants = append(c.chunk.Constants, val)
	return len(c.chunk.Constants) - 1
}

func (c *FluxCompiler) compileBlock(block *ast.BlockExpr) error {
	if block == nil {
		return nil
	}
	for _, expr := range block.Exprs {
		c.compileExpr(expr)
	}
	return nil
}
