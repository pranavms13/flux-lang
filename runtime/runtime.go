// Package runtime evaluates a resolved Flux syntax tree directly.
package runtime

import (
	"fmt"
	"io"
	"math"
	"os"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/resolver"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/value"
)

type Value = any
type BuiltinFunc func(args ...Value) (Value, error)

func (BuiltinFunc) FluxTypeName() string { return "function" }

type Options struct {
	Output io.Writer
	Source *source.Source
	// MaxDepth limits active user function calls. Zero uses 256.
	MaxDepth int
}
type environment map[resolver.ID]*value.Cell
type interpreter struct {
	globals         environment
	bindings        *resolver.Result
	out             io.Writer
	src             *source.Source
	name            string
	depth, maxDepth int
}
type closure struct {
	function *ast.FuncExpr
	captures environment
	name     string
}

func (*closure) FluxTypeName() string { return "function" }
func (c *closure) label() string {
	if c.name == "" {
		return fault.Anonymous
	}
	return c.name
}

func Run(prog *ast.Program, opts Options) error {
	bindings := resolver.Resolve(prog, opts.Source)
	if err := bindings.Failure(opts.Source); err != nil {
		return err
	}
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	limit := opts.MaxDepth
	if limit <= 0 {
		limit = value.DefaultMaxDepth
	}
	r := &interpreter{globals: environment{}, bindings: bindings, out: out, src: opts.Source, maxDepth: limit}
	r.globals[resolver.PrintID] = &value.Cell{Initialized: true, Value: BuiltinFunc(func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fault.ArgumentCount("print", 1, len(args))
		}
		fmt.Fprintln(r.out, value.Display(args[0]))
		return nil, nil
	})}
	for _, stmt := range prog.Statements {
		v, err := r.statement(stmt, environment{})
		if err != nil {
			return err
		}
		if stmt.Expr != nil && v != nil {
			fmt.Fprintln(r.out, value.Display(v))
		}
	}
	return nil
}
func (r *interpreter) locate(at ast.Positioned) source.Location {
	if r.src == nil || at == nil || !at.HasPosition() {
		return source.Location{}
	}
	return r.src.Locate(at.Span(r.src.ID()))
}
func (r *interpreter) fail(err error, at ast.Positioned) error {
	if e, ok := err.(*fault.Error); ok {
		return e.At(r.locate(at))
	}
	return err
}
func (r *interpreter) internal(at ast.Positioned) error {
	return &fault.Error{Code: diagnostic.CodeInternal, Message: "invalid expression or unresolved binding", Where: r.locate(at)}
}
func (r *interpreter) cell(id resolver.ID, local environment) *value.Cell {
	if r.bindings.Bindings[id].Global {
		return r.globals[id]
	}
	return local[id]
}
func (r *interpreter) statement(s *ast.Statement, local environment) (Value, error) {
	if s.Let == nil {
		return r.eval(s.Expr, local)
	}
	id := r.bindings.Declarations[s.Let]
	cell := &value.Cell{}
	if r.bindings.Bindings[id].Global {
		r.globals[id] = cell
	} else {
		local[id] = cell
	}
	oldName := r.name
	r.name = s.Let.Name
	v, err := r.eval(s.Let.Expr, local)
	r.name = oldName
	if err != nil {
		return nil, err
	}
	cell.Value, cell.Initialized = v, true
	return nil, nil
}
func (r *interpreter) eval(n ast.Positioned, local environment) (Value, error) {
	if left, rest := ast.Chain(n); left != nil {
		v, err := r.eval(left, local)
		if err != nil {
			return nil, err
		}
		var at ast.Positioned = left
		for _, op := range rest {
			// Logical operators are control flow, not eager binary operations.
			if op.Operator == "&&" || op.Operator == "||" {
				b, err := value.Bool(v)
				if err != nil {
					return nil, r.fail(err, at)
				}
				if b == (op.Operator == "||") {
					v = b
					at = ast.Range{Left: at, Right: op.At}
					continue
				}
				v, err = r.eval(op.Right, local)
				if err != nil {
					return nil, err
				}
				v, err = value.Bool(v)
				if err != nil {
					return nil, r.fail(err, op.Right)
				}
			} else {
				rhs, err := r.eval(op.Right, local)
				if err != nil {
					return nil, err
				}
				v, err = value.Binary(op.Operator, v, rhs)
				if err != nil {
					var location ast.Positioned = op.At
					if op.Operator == "==" || op.Operator == "!=" {
						location = ast.Range{Left: at, Right: op.At}
					}
					return nil, r.fail(err, location)
				}
			}
			at = ast.Range{Left: at, Right: op.At}
		}
		return v, nil
	}
	switch n := n.(type) {
	case *ast.Expr:
		if n.If != nil {
			return r.eval(n.If, local)
		}
		if n.Func != nil {
			return r.eval(n.Func, local)
		}
		return r.eval(n.Bin, local)
	case *ast.Unary:
		if n.MinLiteral {
			return int64(math.MinInt64), nil
		}
		if n.Primary != nil {
			return r.eval(n.Primary, local)
		}
		v, err := r.eval(n.Operand, local)
		if err != nil {
			return nil, err
		}
		v, err = value.Unary(n.Operator, v)
		return v, r.fail(err, n)
	case *ast.IfExpr:
		v, err := r.eval(n.Cond, local)
		if err != nil {
			return nil, err
		}
		b, err := value.Bool(v)
		if err != nil {
			return nil, r.fail(err, n.Cond)
		}
		if b {
			return r.eval(n.ThenExpr, local)
		}
		return r.eval(n.ElseExpr, local)
	case *ast.FuncExpr:
		captures := environment{}
		for _, id := range r.bindings.Functions[n].Captures {
			captures[id] = r.cell(id, local)
		}
		return &closure{function: n, captures: captures, name: r.name}, nil
	case *ast.BlockExpr:
		inner := copyEnvironment(local)
		var v Value
		for _, s := range n.Statements {
			var err error
			v, err = r.statement(s, inner)
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case *ast.PrimaryExpr:
		v, err := r.eval(n.Base, local)
		if err != nil {
			return nil, err
		}
		for _, p := range n.Postfix {
			if p.Call != nil {
				v, err = r.call(v, p.Call, local)
			} else {
				var index Value
				index, err = r.eval(p.Index.Index, local)
				if err != nil {
					return nil, err
				}
				v, err = value.Index(v, index)
				err = r.fail(err, p.Index)
			}
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case *ast.BaseExpr:
		switch {
		case n.Term != nil:
			return r.eval(n.Term, local)
		case n.Group != nil:
			return r.eval(n.Group.Expr, local)
		case n.Block != nil:
			return r.eval(n.Block, local)
		case n.List != nil:
			return r.eval(n.List, local)
		case n.Dict != nil:
			return r.eval(n.Dict, local)
		}
	case *ast.Term:
		switch {
		case n.Number != nil:
			return n.Number.Value, nil
		case n.String != nil:
			return *n.String, nil
		case n.Bool != nil:
			return bool(*n.Bool), nil
		case n.Ident != nil:
			cell := r.cell(r.bindings.Uses[n], local)
			if cell == nil || !cell.Initialized {
				return nil, r.internal(n)
			}
			return cell.Value, nil
		}
	case *ast.ListExpr:
		values := make([]Value, 0, len(n.Elems))
		for _, e := range n.Elems {
			v, err := r.eval(e, local)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	case *ast.DictExpr:
		dict := map[any]any{}
		for _, p := range n.Pairs {
			k, err := r.eval(p.Key, local)
			if err != nil {
				return nil, err
			}
			v, err := r.eval(p.Value, local)
			if err != nil {
				return nil, err
			}
			if err := value.Key(k); err != nil {
				return nil, r.fail(err, p.Key)
			}
			dict[k] = v
		}
		return dict, nil
	}
	return nil, r.internal(n)
}
func (r *interpreter) call(callee Value, call *ast.CallExpr, local environment) (Value, error) {
	args := make([]Value, 0, len(call.Args))
	for _, a := range call.Args {
		v, err := r.eval(a, local)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	switch fn := callee.(type) {
	case BuiltinFunc:
		v, err := fn(args...)
		return v, r.fail(err, call)
	case *closure:
		if len(args) != len(fn.function.Params) {
			return nil, r.fail(fault.ArgumentCount(fn.label(), len(fn.function.Params), len(args)), call)
		}
		if r.depth >= r.maxDepth {
			return nil, r.fail(fault.CallDepth(r.maxDepth), call)
		}
		env := copyEnvironment(fn.captures)
		for i, p := range fn.function.Params {
			env[r.bindings.Parameters[p]] = &value.Cell{Value: args[i], Initialized: true}
		}
		r.depth++
		oldName := r.name
		r.name = ""
		v, err := r.eval(fn.function.Body, env)
		r.depth--
		r.name = oldName
		if e, ok := err.(*fault.Error); ok {
			return nil, e.WithTrace([]fault.Frame{{Function: fn.label(), Call: r.locate(call)}})
		}
		return v, err
	default:
		return nil, r.fail(fault.NotCallable(callee), call)
	}
}
func copyEnvironment(env environment) environment {
	copy := make(environment, len(env))
	for id, cell := range env {
		copy[id] = cell
	}
	return copy
}
