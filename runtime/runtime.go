// Package runtime evaluates a Flux syntax tree directly.
//
// It is one of two engines, the other being the bytecode VM. Both report
// failures through the shared catalogue in fault, so a program that fails does
// so with the same code, wording, and location whichever engine ran it.
package runtime

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/source"
)

type Value interface{}

// BuiltinFunc is a function provided by the language rather than the program.
type BuiltinFunc func(args ...Value) (Value, error)

// Options configures one execution.
type Options struct {
	// Output receives everything the program prints. It defaults to standard
	// output. Injecting it lets a test read a program's output without
	// replacing a global, which is what forced the execution tests to run one
	// at a time.
	Output io.Writer
	// Source locates failures. Without it a failure still carries its code and
	// message, but cannot say where it happened.
	Source *source.Source
}

type interpreter struct {
	env  map[string]Value
	out  io.Writer
	src  *source.Source
	name string // the identifier a function literal is being bound to
}

type closure struct {
	function *ast.FuncExpr
	locals   map[string]Value
	name     string
}

// label returns the closure name for a call trace, or the anonymous-function
// label when it was never bound.
func (c *closure) label() string {
	if c.name == "" {
		return fault.Anonymous
	}
	return c.name
}

// Run executes a program in an environment of its own, returning the first
// failure it hits.
//
// Failures are returned rather than raised, so that a mistake in the program
// stays distinguishable from a defect in the interpreter.
func Run(prog *ast.Program, opts Options) error {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	r := &interpreter{env: map[string]Value{}, out: out, src: opts.Source}
	r.env["print"] = BuiltinFunc(func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fault.ArgumentCount("print", 1, len(args))
		}
		fmt.Fprintln(r.out, args[0])
		return nil, nil
	})
	for _, stmt := range prog.Statements {
		if stmt.Let != nil {
			r.name = stmt.Let.Name
			value, err := r.evalExpr(stmt.Let.Expr, nil)
			r.name = ""
			if err != nil {
				return err
			}
			r.env[stmt.Let.Name] = value
			continue
		}
		value, err := r.evalExpr(stmt.Expr, nil)
		if err != nil {
			return err
		}
		if value != nil {
			fmt.Fprintln(r.out, value)
		}
	}
	return nil
}

// locate resolves a node's position, or returns the zero location when the
// interpreter was given no source.
func (r *interpreter) locate(at ast.Positioned) source.Location {
	if r.src == nil || at == nil || !at.HasPosition() {
		return source.Location{}
	}
	return r.src.Locate(at.Span(r.src.ID()))
}

// fail attaches the position of the construct that failed to a catalogue entry.
func (r *interpreter) fail(err error, at ast.Positioned) error {
	if failure, ok := err.(*fault.Error); ok {
		return failure.At(r.locate(at))
	}
	return err
}

// internal reports a defect in the interpreter. A user program must not be able
// to reach one; the separate code keeps the CLI from presenting it as a mistake
// in the program.
func (r *interpreter) internal(at ast.Positioned, format string, args ...any) error {
	return &fault.Error{
		Code:    diagnostic.CodeInternal,
		Message: fmt.Sprintf(format, args...),
		Where:   r.locate(at),
	}
}

// evalExpr evaluates an expression in the current locals and returns the first
// located failure.
func (r *interpreter) evalExpr(expr *ast.Expr, local map[string]Value) (Value, error) {
	switch {
	case expr.If != nil:
		cond, err := r.evalExpr(expr.If.Cond, local)
		if err != nil {
			return nil, err
		}
		if truthy(cond) {
			return r.evalExpr(expr.If.ThenExpr, local)
		}
		return r.evalExpr(expr.If.ElseExpr, local)
	case expr.Bin != nil:
		value, err := r.evalAdditive(expr.Bin.Left, local)
		if err != nil {
			return nil, err
		}
		for _, rest := range expr.Bin.Rest {
			right, err := r.evalAdditive(rest.Right, local)
			if err != nil {
				return nil, err
			}
			if value, err = r.apply(rest, rest.Operator, value, right); err != nil {
				return nil, err
			}
		}
		return value, nil
	case expr.Block != nil:
		return r.evalBlock(expr.Block, local)
	case expr.Primary != nil:
		return r.evalPrimary(expr.Primary, local)
	case expr.Func != nil:
		return &closure{function: expr.Func, locals: copyLocals(local), name: r.name}, nil
	default:
		return nil, r.internal(expr, "expression matched no grammar alternative")
	}
}

// evalPrimary evaluates a base value and then applies calls and indexes in
// source order.
func (r *interpreter) evalPrimary(primary *ast.PrimaryExpr, local map[string]Value) (Value, error) {
	val, err := r.evalBase(primary.Base, local)
	if err != nil {
		return nil, err
	}
	for _, pf := range primary.Postfix {
		switch {
		case pf.Call != nil:
			if val, err = r.evalCall(val, pf.Call, local); err != nil {
				return nil, err
			}
		case pf.Index != nil:
			index, err := r.evalExpr(pf.Index.Index, local)
			if err != nil {
				return nil, err
			}
			result, err := indexValue(val, index)
			if err != nil {
				return nil, r.fail(err, pf.Index)
			}
			val = result
		}
	}
	return val, nil
}

// evalBase evaluates a literal, group, block, or collection before any postfix
// operations.
func (r *interpreter) evalBase(base *ast.BaseExpr, local map[string]Value) (Value, error) {
	switch {
	case base == nil:
		return nil, nil
	case base.Term != nil:
		return r.evalTerm(base.Term, local)
	case base.Group != nil:
		return r.evalExpr(base.Group.Expr, local)
	case base.Block != nil:
		return r.evalBlock(base.Block, local)
	case base.List != nil:
		values := make([]Value, 0, len(base.List.Elems))
		for _, elem := range base.List.Elems {
			value, err := r.evalExpr(elem, local)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	case base.Dict != nil:
		dict := make(map[interface{}]interface{}, len(base.Dict.Pairs))
		for _, pair := range base.Dict.Pairs {
			key, err := r.evalExpr(pair.Key, local)
			if err != nil {
				return nil, err
			}
			value, err := r.evalExpr(pair.Value, local)
			if err != nil {
				return nil, err
			}
			dict[key] = value
		}
		return dict, nil
	}
	return nil, r.internal(base, "base expression matched no grammar alternative")
}

// evalCall evaluates arguments, checks callability and arity, and adds the
// call site to failures from the function body.
func (r *interpreter) evalCall(callee Value, call *ast.CallExpr, local map[string]Value) (Value, error) {
	args := make([]Value, 0, len(call.Args))
	for _, argExpr := range call.Args {
		arg, err := r.evalExpr(argExpr, local)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	switch fn := callee.(type) {
	case BuiltinFunc:
		value, err := fn(args...)
		if err != nil {
			return nil, r.fail(err, call)
		}
		return value, nil
	case *closure:
		if len(fn.function.Params) != len(args) {
			return nil, r.fail(fault.ArgumentCount(fn.label(), len(fn.function.Params), len(args)), call)
		}
		localEnv := copyLocals(fn.locals)
		for i, param := range fn.function.Params {
			localEnv[param.Name] = args[i]
		}
		value, err := r.evalExpr(fn.function.Body, localEnv)
		if err != nil {
			// The failure unwinds through this call, so the frame naming the
			// function and the call site is added here, innermost first.
			return nil, addFrame(err, fn.label(), r.locate(call))
		}
		return value, nil
	default:
		return nil, r.fail(fault.NotCallable(callee), call)
	}
}

// evalTerm evaluates a literal or resolves a name from locals before globals,
// reporting an undefined value at its use.
func (r *interpreter) evalTerm(term *ast.Term, local map[string]Value) (Value, error) {
	switch {
	case term.Bool != nil:
		return bool(*term.Bool), nil
	case term.Number != nil:
		return *term.Number, nil
	case term.String != nil:
		return *term.String, nil
	case term.Ident != nil:
		if local != nil {
			if val, ok := local[*term.Ident]; ok {
				return val, nil
			}
		}
		val, ok := r.env[*term.Ident]
		if !ok {
			return nil, r.fail(fault.UndefinedValue(*term.Ident), term)
		}
		return val, nil
	}
	return nil, r.internal(term, "term matched no grammar alternative")
}

// evalBlock evaluates expressions in order and returns the last value, or nil
// when the block is absent or empty.
func (r *interpreter) evalBlock(block *ast.BlockExpr, local map[string]Value) (Value, error) {
	if block == nil {
		return nil, nil
	}
	var result Value
	for _, expr := range block.Exprs {
		value, err := r.evalExpr(expr, local)
		if err != nil {
			return nil, err
		}
		result = value
	}
	return result, nil
}

// evalAdditive evaluates addition and subtraction left to right, locating
// failures at the operator application.
func (r *interpreter) evalAdditive(expr *ast.Additive, local map[string]Value) (Value, error) {
	value, err := r.evalPrimary(expr.Left, local)
	if err != nil {
		return nil, err
	}
	for _, rest := range expr.Rest {
		right, err := r.evalPrimary(rest.Right, local)
		if err != nil {
			return nil, err
		}
		if value, err = r.apply(rest, rest.Operator, value, right); err != nil {
			return nil, err
		}
	}
	return value, nil
}

// apply evaluates a binary operation and attaches the AST location to any
// catalogue failure.
func (r *interpreter) apply(at ast.Positioned, operator string, left, right Value) (Value, error) {
	value, err := binaryValue(operator, left, right)
	if err != nil {
		return nil, r.fail(err, at)
	}
	return value, nil
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

func copyLocals(local map[string]Value) map[string]Value {
	result := make(map[string]Value, len(local))
	for name, value := range local {
		result[name] = value
	}
	return result
}

// indexValue applies an index, matching the VM's implementation of the same
// operation. Both report through the same catalogue entries.
func indexValue(value, index Value) (Value, error) {
	switch v := value.(type) {
	case []Value:
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

// binaryValue implements supported binary operators and reports incompatible
// operands through the shared fault catalogue.
func binaryValue(operator string, left, right Value) (Value, error) {
	switch operator {
	case "+":
		switch value := left.(type) {
		case int:
			if other, ok := right.(int); ok {
				return value + other, nil
			}
		case string:
			if other, ok := right.(string); ok {
				return value + other, nil
			}
		}
	case "-", ">", "<":
		a, aOK := left.(int)
		b, bOK := right.(int)
		if aOK && bOK {
			switch operator {
			case "-":
				return a - b, nil
			case ">":
				return a > b, nil
			default:
				return a < b, nil
			}
		}
	case "==":
		return reflect.DeepEqual(left, right), nil
	}
	return nil, fault.OperandType(operator, left, right)
}

// addFrame adds a call site to a runtime fault while preserving other error
// types.
func addFrame(err error, function string, call source.Location) error {
	if failure, ok := err.(*fault.Error); ok {
		return failure.WithTrace([]fault.Frame{{Function: function, Call: call}})
	}
	return err
}
