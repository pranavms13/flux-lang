package resolver_test

import (
	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/resolver"
	"github.com/pranavms13/flux-lang/source"
	"testing"
)

func TestBindingIdentityAndTransitiveCaptures(t *testing.T) {
	src := source.New(1, "scope.flux", `let x=1;let make=fn(p: int)=>{let before=fn()=>x;let x=p;fn()=>fn()=>x};print(make(2)()())`)
	parsed := parser.ParseSource(src)
	if parsed.Failed() {
		t.Fatal(parsed.Diagnostics)
	}
	resolved := resolver.Resolve(parsed.Program, src)
	if len(resolved.Diagnostics) != 0 {
		t.Fatal(resolved.Diagnostics)
	}
	global := resolved.Declarations[parsed.Program.Statements[0].Let]
	var local resolver.ID
	for _, b := range resolved.Bindings {
		if b.Name == "x" && !b.Global {
			local = b.ID
		}
	}
	if local == 0 || local == global {
		t.Fatal("local and global binding identities must differ")
	}
	captured := 0
	globalUses, localUses := 0, 0
	for _, f := range resolved.Functions {
		for _, id := range f.Captures {
			if id == local {
				captured++
			}
		}
	}
	for term, id := range resolved.Uses {
		if *term.Ident == "x" {
			if id == global {
				globalUses++
			}
			if id == local {
				localUses++
			}
		}
	}
	if captured != 2 || globalUses != 1 || localUses != 1 {
		t.Fatalf("captures=%d global uses=%d local uses=%d", captured, globalUses, localUses)
	}
	// Running resolution twice must not change the AST or IDs.
	again := resolver.Resolve(parsed.Program, src)
	ast.Walk(parsed.Program, func(n ast.Positioned) bool {
		if term, ok := n.(*ast.Term); ok && term.Ident != nil {
			if resolved.Uses[term] != again.Uses[term] {
				t.Fatal("unstable identity")
			}
		}
		return true
	})
	if len(resolved.Scopes) < 5 {
		t.Fatal("scope metadata missing")
	}
}
func TestDuplicateRelatedLocation(t *testing.T) {
	src := source.New(1, "duplicate.flux", "let x=1;let x=2")
	result := resolver.Resolve(parser.ParseSource(src).Program, src)
	if len(result.Diagnostics) != 1 {
		t.Fatal(result.Diagnostics)
	}
	d := result.Diagnostics[0]
	if d.Code != resolver.CodeDuplicateDeclaration || src.TextOf(d.Primary) != "let x=2" || len(d.Related) != 1 || src.TextOf(d.Related[0].Span) != "let x=1" {
		t.Fatal(d)
	}
}
