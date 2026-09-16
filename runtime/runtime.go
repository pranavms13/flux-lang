package runtime

import (
	"fmt"
	"reflect"

	"github.com/pranavms13/flux-lang/ast"
)

type Value interface{}
type BuiltinFunc func(args ...Value) Value

type interpreter struct{ env map[string]Value }

type closure struct {
	function *ast.FuncExpr
	locals   map[string]Value
}

// Run creates an isolated environment for each program.
func Run(prog *ast.Program) {
	r := &interpreter{env: map[string]Value{}}
	r.env["print"] = BuiltinFunc(func(args ...Value) Value {
		if len(args) != 1 {
			panic("print expects 1 argument")
		}
		fmt.Println(args[0])
		return nil
	})
	for _, stmt := range prog.Statements {
		if stmt.Let != nil {
			r.env[stmt.Let.Name] = r.evalExpr(stmt.Let.Expr, nil)
		} else if value := r.evalExpr(stmt.Expr, nil); value != nil {
			fmt.Println(value)
		}
	}
}

func (r *interpreter) evalExpr(expr *ast.Expr, local map[string]Value) Value {
	switch {
	case expr.If != nil:
		cond := r.evalExpr(expr.If.Cond, local)
		if truthy(cond) {
			return r.evalExpr(expr.If.ThenExpr, local)
		}
		return r.evalExpr(expr.If.ElseExpr, local)
	case expr.Bin != nil:
		value := r.evalAdditive(expr.Bin.Left, local)
		for _, rest := range expr.Bin.Rest {
			value = binaryValue(rest.Operator, value, r.evalAdditive(rest.Right, local))
		}
		return value
	case expr.Block != nil:
		return r.evalBlock(expr.Block, local)
	case expr.Primary != nil:
		// Evaluate the base
		var val Value
		if expr.Primary.Base != nil {
			if expr.Primary.Base.Term != nil {
				val = r.evalTerm(expr.Primary.Base.Term, local)
			} else if expr.Primary.Base.Group != nil {
				val = r.evalExpr(expr.Primary.Base.Group.Expr, local)
			} else if expr.Primary.Base.Block != nil {
				val = r.evalBlock(expr.Primary.Base.Block, local)
			} else if expr.Primary.Base.List != nil {
				vals := []Value{}
				for _, e := range expr.Primary.Base.List.Elems {
					vals = append(vals, r.evalExpr(e, local))
				}
				val = vals
			} else if expr.Primary.Base.Dict != nil {
				dict := make(map[interface{}]interface{})
				for _, pair := range expr.Primary.Base.Dict.Pairs {
					key := r.evalExpr(pair.Key, local)
					value := r.evalExpr(pair.Value, local)
					dict[key] = value
				}
				val = dict
			}
		}
		// Apply postfixes
		for _, pf := range expr.Primary.Postfix {
			if pf.Call != nil {
				// Function call
				fnVal := val
				var args []Value
				for _, argExpr := range pf.Call.Args {
					args = append(args, r.evalExpr(argExpr, local))
				}
				if builtin, ok := fnVal.(BuiltinFunc); ok {
					val = builtin(args...)
				} else if fn, ok := fnVal.(*closure); ok {
					if len(fn.function.Params) != len(args) {
						panic("argument count mismatch")
					}
					localEnv := copyLocals(fn.locals)
					for i, param := range fn.function.Params {
						localEnv[param.Name] = args[i]
					}
					val = r.evalExpr(fn.function.Body, localEnv)
				} else {
					panic("not a function")
				}
			} else if pf.Index != nil {
				indexVal := r.evalExpr(pf.Index.Index, local)
				switch v := val.(type) {
				case []Value:
					// Array indexing
					idx, ok := indexVal.(int)
					if !ok {
						panic("Array index must be an integer")
					}
					if idx < 0 || idx >= len(v) {
						panic("Array index out of bounds")
					}
					val = v[idx]
				case map[interface{}]interface{}:
					// Dictionary access
					value, exists := v[indexVal]
					if !exists {
						panic(fmt.Sprintf("Key not found in dictionary: %v", indexVal))
					}
					val = value
				default:
					panic(fmt.Sprintf("Cannot index into value of type %T", val))
				}
			}
		}
		return val
	case expr.Func != nil:
		return &closure{function: expr.Func, locals: copyLocals(local)}
	default:
		panic("unknown expression")
	}
}

func (r *interpreter) evalTerm(term *ast.Term, local map[string]Value) Value {
	if term.Bool != nil {
		return bool(*term.Bool)
	} else if term.Number != nil {
		return *term.Number
	} else if term.String != nil {
		return *term.String
	} else if term.Ident != nil {
		if local != nil {
			if val, ok := local[*term.Ident]; ok {
				return val
			}
		}
		val, ok := r.env[*term.Ident]
		if !ok {
			panic("undefined variable: " + *term.Ident)
		}
		return val
	}
	panic("invalid term")
}

func truthy(val Value) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case string:
		return v != ""
	default:
		return val != nil
	}
}

func (r *interpreter) evalBlock(block *ast.BlockExpr, local map[string]Value) Value {
	if block == nil {
		return nil
	}
	var result Value
	for _, expr := range block.Exprs {
		result = r.evalExpr(expr, local)
	}
	return result
}

func copyLocals(local map[string]Value) map[string]Value {
	result := make(map[string]Value, len(local))
	for name, value := range local {
		result[name] = value
	}
	return result
}

func (r *interpreter) evalAdditive(expr *ast.Additive, local map[string]Value) Value {
	value := r.evalExpr(&ast.Expr{Primary: expr.Left}, local)
	for _, rest := range expr.Rest {
		value = binaryValue(rest.Operator, value, r.evalExpr(&ast.Expr{Primary: rest.Right}, local))
	}
	return value
}

func binaryValue(operator string, left, right Value) Value {
	switch operator {
	case "+":
		switch value := left.(type) {
		case int:
			return value + right.(int)
		case string:
			return value + right.(string)
		}
	case "-":
		return left.(int) - right.(int)
	case "==":
		return reflect.DeepEqual(left, right)
	case ">":
		return left.(int) > right.(int)
	case "<":
		return left.(int) < right.(int)
	}
	panic("unsupported operator or operands: " + operator)
}
