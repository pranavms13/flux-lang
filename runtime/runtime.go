package runtime

import (
	"fmt"

	"github.com/pranavms13/flux-lang/parser"
)

type Value interface{}
type BuiltinFunc func(args ...Value) Value

var env = map[string]Value{}

func init() {
	env["print"] = BuiltinFunc(func(args ...Value) Value {
		for _, arg := range args {
			fmt.Println(arg)
		}
		return nil
	})
}

func Run(prog *parser.Program) {
	for _, stmt := range prog.Statements {
		runStatement(stmt)
	}
}

func runStatement(stmt *parser.Statement) {
	if stmt.Let != nil {
		val := evalExpr(stmt.Let.Expr, nil)
		env[stmt.Let.Name] = val
	} else if stmt.Expr != nil {
		if call := stmt.Expr.Call; call != nil && call.Name == "print" {
			evalExpr(stmt.Expr, nil) // don't print print's return value
		} else {
			val := evalExpr(stmt.Expr, nil)
			fmt.Println(val)
		}
	}
}

func evalExpr(expr *parser.Expr, local map[string]Value) Value {
	switch {
	case expr.If != nil:
		cond := evalExpr(expr.If.Cond, local)
		if truthy(cond) {
			return evalExpr(expr.If.ThenExpr, local)
		}
		return evalExpr(expr.If.ElseExpr, local)
	case expr.Func != nil:
		return expr.Func
	case expr.Call != nil:
		fnVal, ok := env[expr.Call.Name]
		if !ok {
			panic("undefined function: " + expr.Call.Name)
		}

		// Handle built-in functions
		if builtin, ok := fnVal.(BuiltinFunc); ok {
			var args []Value
			for _, argExpr := range expr.Call.Args {
				args = append(args, evalExpr(argExpr, local))
			}
			return builtin(args...)
		}

		funcExpr, ok := fnVal.(*parser.FuncExpr)
		if !ok {
			panic("not a function: " + expr.Call.Name)
		}
		if len(funcExpr.Params) != len(expr.Call.Args) {
			panic("argument count mismatch")
		}
		localEnv := make(map[string]Value)
		for i, param := range funcExpr.Params {
			localEnv[param] = evalExpr(expr.Call.Args[i], local)
		}
		return evalExpr(funcExpr.Body, localEnv)

	case expr.Bin != nil:
		left := evalTerm(expr.Bin.Left, local)
		if expr.Bin.Operator == nil || expr.Bin.Right == nil {
			return left
		}
		right := evalExpr(expr.Bin.Right, local)
		// support int or string concatenation
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
		var result Value
		for _, e := range expr.Block.Exprs {
			result = evalExpr(e, local) // ← pass `local` here
		}
		return result

	case expr.Term != nil:
		return evalTerm(expr.Term, local)

	default:
		panic("unknown expression")
	}
}

func evalTerm(term *parser.Term, local map[string]Value) Value {
	if term.Bool != nil {
		return *term.Bool
	} else if term.Number != nil {
		return *term.Number
	} else if term.String != nil {
		return *term.String
	} else if term.Ident != nil {
		// First check local environment
		if local != nil {
			if val, ok := local[*term.Ident]; ok {
				return val
			}
		}
		// Then check global environment
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
	case int:
		return v != 0
	case string:
		return v != ""
	default:
		return val != nil
	}
}
