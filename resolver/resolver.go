// Package resolver assigns lexical identities independently of type checking.
package resolver

import (
	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/source"
)

type ID uint32

const PrintID ID = 1

var (
	CodeUndefined            = diagnostic.Register("B_UNDEFINED_VARIABLE", "a name was used where nothing declares it")
	CodeDuplicateParameter   = diagnostic.Register("B_DUPLICATE_PARAMETER", "a function declares the same parameter name twice")
	CodeDuplicateDeclaration = diagnostic.Register("B_DUPLICATE_DECLARATION", "a scope declares the same name twice")
	CodeSelfInitialization   = diagnostic.Register("B_SELF_INITIALIZATION", "a binding reads itself before initialization")
)

type Binding struct {
	ID          ID
	Name        string
	Declaration ast.Positioned
	Scope       int
	Function    *ast.FuncExpr
	Global      bool
}
type Scope struct {
	Parent      int
	Declaration ast.Positioned
	Bindings    []ID
}
type Function struct{ Captures []ID }
type Result struct {
	Bindings     map[ID]Binding
	Scopes       []Scope
	Declarations map[*ast.LetStatement]ID
	Parameters   map[*ast.FuncParam]ID
	Uses         map[*ast.Term]ID
	Functions    map[*ast.FuncExpr]*Function
	Recursive    map[*ast.LetStatement]bool
	Diagnostics  []diagnostic.Diagnostic
}

// Failure supports low-level execution APIs that were not preceded by a check.
// CLI/checker consumers use Diagnostics directly, retaining related locations.
func (r *Result) Failure(src *source.Source) *fault.Error {
	if len(r.Diagnostics) == 0 {
		return nil
	}
	d := r.Diagnostics[0]
	e := &fault.Error{Code: d.Code, Message: d.Message}
	if src != nil {
		e.Where = src.Locate(d.Primary)
	}
	return e
}

type declaration struct {
	id           ID
	initializing bool
	recursive    *ast.LetStatement
}
type resolver struct {
	result   *Result
	src      *source.Source
	scopes   []map[string]*declaration
	scopeIDs []int
	funcs    []*ast.FuncExpr
	next     ID
}

func Resolve(prog *ast.Program, src *source.Source) *Result {
	result := &Result{Bindings: map[ID]Binding{}, Declarations: map[*ast.LetStatement]ID{}, Parameters: map[*ast.FuncParam]ID{}, Uses: map[*ast.Term]ID{}, Functions: map[*ast.FuncExpr]*Function{}, Recursive: map[*ast.LetStatement]bool{}}
	r := &resolver{result: result, src: src, next: PrintID}
	r.enter(prog)
	id := r.declare("print", nil, false)
	b := result.Bindings[id]
	b.Global = true
	result.Bindings[id] = b
	r.enter(prog)
	r.walk(prog)
	return result
}
func (r *resolver) span(n ast.Positioned) source.Span {
	if r.src == nil || n == nil {
		return source.NoSpan
	}
	return n.Span(r.src.ID())
}
func (r *resolver) currentFunction() *ast.FuncExpr {
	if len(r.funcs) == 0 {
		return nil
	}
	return r.funcs[len(r.funcs)-1]
}
func (r *resolver) enter(n ast.Positioned) {
	parent := -1
	if len(r.scopeIDs) > 0 {
		parent = r.scopeIDs[len(r.scopeIDs)-1]
	}
	r.scopeIDs = append(r.scopeIDs, len(r.result.Scopes))
	r.result.Scopes = append(r.result.Scopes, Scope{Parent: parent, Declaration: n})
	r.scopes = append(r.scopes, map[string]*declaration{})
}
func (r *resolver) leave() {
	r.scopes = r.scopes[:len(r.scopes)-1]
	r.scopeIDs = r.scopeIDs[:len(r.scopeIDs)-1]
}
func (r *resolver) declare(name string, n ast.Positioned, param bool) ID {
	scope := r.scopes[len(r.scopes)-1]
	if first, ok := scope[name]; ok {
		code, label := CodeDuplicateDeclaration, "declaration"
		if param {
			code, label = CodeDuplicateParameter, "parameter"
		}
		d := diagnostic.Error(code, r.span(n), "duplicate %s: %s", label, name)
		if span := r.span(r.result.Bindings[first.id].Declaration); span.IsValid() {
			d = d.WithRelated(span, "%s is already declared here", name)
		}
		r.result.Diagnostics = append(r.result.Diagnostics, d)
	}
	id := r.next
	r.next++
	scope[name] = &declaration{id: id}
	sid := r.scopeIDs[len(r.scopeIDs)-1]
	r.result.Bindings[id] = Binding{ID: id, Name: name, Declaration: n, Scope: sid, Function: r.currentFunction(), Global: len(r.scopes) == 2}
	r.result.Scopes[sid].Bindings = append(r.result.Scopes[sid].Bindings, id)
	return id
}
func (r *resolver) walk(n ast.Positioned) {
	ast.Walk(n, func(n ast.Positioned) bool {
		switch n := n.(type) {
		case *ast.LetStatement:
			id := r.declare(n.Name, n, false)
			r.result.Declarations[n] = id
			d := r.scopes[len(r.scopes)-1][n.Name]
			d.initializing = true
			if ast.FunctionLiteral(n.Expr) != nil {
				d.recursive = n
			}
			r.walk(n.Expr)
			d.initializing = false
			return false
		case *ast.FuncExpr:
			r.result.Functions[n] = &Function{}
			r.funcs = append(r.funcs, n)
			r.enter(n)
			for _, p := range n.Params {
				r.result.Parameters[p] = r.declare(p.Name, p, true)
			}
			r.walk(n.Body)
			r.leave()
			r.funcs = r.funcs[:len(r.funcs)-1]
			return false
		case *ast.BlockExpr:
			r.enter(n)
			for _, s := range n.Statements {
				r.walk(s)
			}
			r.leave()
			return false
		case *ast.Term:
			if n.Ident != nil {
				r.use(n)
			}
		}
		return true
	})
}
func (r *resolver) use(n *ast.Term) {
	name := *n.Ident
	var d *declaration
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if found, ok := r.scopes[i][name]; ok {
			d = found
			break
		}
	}
	if d == nil {
		r.result.Diagnostics = append(r.result.Diagnostics, diagnostic.Error(CodeUndefined, r.span(n), "undefined variable: %s", name))
		return
	}
	b := r.result.Bindings[d.id]
	if d.initializing {
		if d.recursive == nil || r.currentFunction() == b.Function {
			r.result.Diagnostics = append(r.result.Diagnostics, diagnostic.Error(CodeSelfInitialization, r.span(n), "cannot read %s in its own initializer", name))
			return
		}
		r.result.Recursive[d.recursive] = true
	}
	r.result.Uses[n] = d.id
	if b.Global {
		return
	}
	// Propagate captures through intermediate functions, so a grandchild can
	// capture a cell after both enclosing calls have returned.
	for i := len(r.funcs) - 1; i >= 0; i-- {
		f := r.funcs[i]
		if f == b.Function {
			break
		}
		info := r.result.Functions[f]
		found := false
		for _, id := range info.Captures {
			if id == d.id {
				found = true
				break
			}
		}
		if !found {
			info.Captures = append(info.Captures, d.id)
		}
	}
}
