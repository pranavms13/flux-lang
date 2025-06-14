package runtime

import (
	"fmt"

	"github.com/pranavms13/flux-lang/ast"
)

type Value interface{}
type BuiltinFunc func(args ...Value) Value

var env = map[string]Value{}

func init() {
	env["print"] = BuiltinFunc(func(args ...Value) Value {
		if len(args) == 0 {
			return nil
		}
		// Just return the last argument without printing
		return args[len(args)-1]
	})
}

func Run(prog *ast.Program) {
	for _, stmt := range prog.Statements {
		runStatement(stmt)
	}
}

func runStatement(stmt *ast.Statement) {
	if stmt.Let != nil {
		val := evalExpr(stmt.Let.Expr, nil)
		env[stmt.Let.Name] = val
	} else if stmt.Expr != nil {
		// Check if this is a print call before evaluating
		isPrint := false
		if stmt.Expr.Primary != nil && stmt.Expr.Primary.Base != nil && stmt.Expr.Primary.Base.Term != nil && stmt.Expr.Primary.Base.Term.Ident != nil && *stmt.Expr.Primary.Base.Term.Ident == "print" {
			if len(stmt.Expr.Primary.Postfix) > 0 && stmt.Expr.Primary.Postfix[0].Call != nil {
				isPrint = true
			}
		}

		// Check if this is an array indexing or dictionary access
		isIndexing := false
		if stmt.Expr.Primary != nil && len(stmt.Expr.Primary.Postfix) > 0 && stmt.Expr.Primary.Postfix[0].Index != nil {
			isIndexing = true
		}

		val := evalExpr(stmt.Expr, nil)
		// Only print if it's not a print call and not an array indexing
		if !isPrint && !isIndexing {
			fmt.Println(val)
		}
	}
}

func evalExpr(expr *ast.Expr, local map[string]Value) Value {
	switch {
	case expr.If != nil:
		cond := evalExpr(expr.If.Cond, local)
		if truthy(cond) {
			return evalExpr(expr.If.ThenExpr, local)
		}
		return evalExpr(expr.If.ElseExpr, local)
	case expr.Bin != nil:
		left := evalExpr(&ast.Expr{Primary: expr.Bin.Left}, local)
		if expr.Bin.Operator == nil || expr.Bin.Right == nil {
			return left
		}
		right := evalExpr(expr.Bin.Right, local)
		switch *expr.Bin.Operator {
		case "+":
			switch l := left.(type) {
			case int:
				return l + right.(int)
			case string:
				return l + right.(string)
			default:
				panic("unsupported + operands")
			}
		case "-":
			return left.(int) - right.(int)
		case "==":
			return left == right
		case ">":
			return left.(int) > right.(int)
		case "<":
			return left.(int) < right.(int)
		default:
			panic("unsupported operator: " + *expr.Bin.Operator)
		}
	case expr.Block != nil:
		return evalBlock(expr.Block, local)
	case expr.Primary != nil:
		// Evaluate the base
		var val Value
		if expr.Primary.Base != nil {
			if expr.Primary.Base.Term != nil {
				val = evalTerm(expr.Primary.Base.Term, local)
			} else if expr.Primary.Base.List != nil {
				vals := []Value{}
				for _, e := range expr.Primary.Base.List.Elems {
					vals = append(vals, evalExpr(e, local))
				}
				val = vals
			} else if expr.Primary.Base.Dict != nil {
				dict := make(map[interface{}]interface{})
				for _, pair := range expr.Primary.Base.Dict.Pairs {
					key := evalExpr(pair.Key, local)
					value := evalExpr(pair.Value, local)
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
					args = append(args, evalExpr(argExpr, local))
				}
				// If val is a string (function name), look up in env
				if name, ok := fnVal.(string); ok {
					fnVal, ok = env[name]
					if !ok {
						panic("undefined function: " + name)
					}
				}
				if builtin, ok := fnVal.(BuiltinFunc); ok {
					val = builtin(args...)
				} else if funcExpr, ok := fnVal.(*ast.FuncExpr); ok {
					if len(funcExpr.Params) != len(args) {
						panic("argument count mismatch")
					}
					localEnv := make(map[string]Value)
					for i, param := range funcExpr.Params {
						localEnv[param.Name] = args[i]
					}
					val = evalExpr(funcExpr.Body, localEnv)
				} else {
					panic("not a function")
				}
			} else if pf.Index != nil {
				indexVal := evalExpr(pf.Index.Index, local)
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
		return expr.Func
	default:
		panic("unknown expression")
	}
}

func evalTerm(term *ast.Term, local map[string]Value) Value {
	if term.Bool != nil {
		return *term.Bool
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
		val, ok := env[*term.Ident]
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

func evalBlock(block *ast.BlockExpr, local map[string]Value) Value {
	if block == nil {
		return nil
	}
	var result Value
	for _, expr := range block.Exprs {
		result = evalExpr(expr, local)
	}
	return result
}
