package types_test

import (
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
)

// check parses and type checks a program, returning the checker and the source
// so that a test can assert on what a diagnostic points at.
func check(t *testing.T, text string, mode types.TypeCheckingMode) (*types.TypeChecker, *source.Source) {
	t.Helper()
	result := parser.ParseSource(source.New(1, "check.flux", text))
	if result.Failed() {
		t.Fatalf("parse %q: %v", text, result.Diagnostics)
	}
	checker := types.NewTypeCheckerForSource(result.Source, mode)
	checker.CheckProgram(result.Program)
	return checker, result.Source
}

func strict() types.TypeCheckingMode {
	return types.TypeCheckingMode{Enabled: true, Strict: true}
}

// only returns the single diagnostic a program is expected to produce. Several
// diagnostics usually means a cascade, which would make the assertion below
// depend on which one happened to come first.
func only(t *testing.T, checker *types.TypeChecker) diagnostic.Diagnostic {
	t.Helper()
	all := checker.Diagnostics()
	if len(all) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %v", len(all), checker.GetErrors())
	}
	return all[0]
}

// TestDiagnosticsPointAtTheOffendingConstruct is the core of P1.3: a checker
// that reports the right message at the wrong place is no better than one that
// reports no place at all.
func TestDiagnosticsPointAtTheOffendingConstruct(t *testing.T) {
	for _, test := range []struct {
		name    string
		source  string
		code    diagnostic.Code
		primary string
		related string
	}{
		{
			name:    "annotation mismatch points at the value",
			source:  `let x: int = "hello"`,
			code:    types.CodeAnnotationMismatch,
			primary: `"hello"`,
			related: ": int",
		},
		{
			name:    "argument mismatch points at the argument",
			source:  `let f = fn(a: int) => a f("wrong")`,
			code:    types.CodeArgumentType,
			primary: `"wrong"`,
			related: "a: int",
		},
		{
			name:    "return mismatch points at the body",
			source:  `let f = fn(): int => "wrong"`,
			code:    types.CodeReturnMismatch,
			primary: `"wrong"`,
			related: ": int",
		},
		{
			name:    "branch mismatch points at the else branch",
			source:  `let x = if true then { 2 } else { "wrong" }`,
			code:    types.CodeBranchMismatch,
			primary: `{ "wrong" }`,
			related: "{ 2 }",
		},
		{
			name:    "list element points at the element",
			source:  `let xs = [1, "wrong"]`,
			code:    types.CodeListElementType,
			primary: `"wrong"`,
			related: "1",
		},
		{
			name:    "dictionary value points at the value",
			source:  `let d = {"a": 1, "b": "wrong"}`,
			code:    types.CodeDictValueType,
			primary: `"wrong"`,
			related: "1",
		},
		{
			name:    "arity points at the argument list",
			source:  `let f = fn(a: int) => a f()`,
			code:    types.CodeArgumentCount,
			primary: `()`,
			related: "f",
		},
		{
			name:    "undefined name points at the name",
			source:  `print(missing)`,
			code:    types.CodeUndefinedVariable,
			primary: "missing",
		},
		{
			name:    "duplicate parameter points at the repeat",
			source:  `let f = fn(x: int, x: int) => x`,
			code:    types.CodeDuplicateParameter,
			primary: "x: int",
			related: "x: int",
		},
		{
			name:    "index type points at the index",
			source:  `let xs = [1] print(xs["wrong"])`,
			code:    types.CodeIndexType,
			primary: `"wrong"`,
		},
		{
			name:    "condition points at the condition",
			source:  `print(if 1 then { 2 } else { 3 })`,
			code:    types.CodeConditionType,
			primary: "1",
		},
		{
			name:    "invalid dictionary key points at the key",
			source:  `let d = {[1]: 2}`,
			code:    types.CodeInvalidDictKey,
			primary: "[1]",
		},
		{
			name:    "operand type covers both operands",
			source:  `print(1 + "wrong")`,
			code:    types.CodeOperandType,
			primary: `1 + "wrong"`,
			related: `"wrong"`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			checker, src := check(t, test.source, strict())
			d := only(t, checker)

			if d.Code != test.code {
				t.Fatalf("code = %q, want %q", d.Code, test.code)
			}
			if got := src.TextOf(d.Primary); got != test.primary {
				t.Errorf("points at %q, want %q", got, test.primary)
			}
			if test.related == "" {
				if len(d.Related) != 0 {
					t.Errorf("unexpected related location %q", src.TextOf(d.Related[0].Span))
				}
				return
			}
			if len(d.Related) == 0 {
				t.Fatalf("no related location; want one covering %q", test.related)
			}
			if got := src.TextOf(d.Related[0].Span); got != test.related {
				t.Errorf("related location covers %q, want %q", got, test.related)
			}
			// Two occurrences of one identifier read identically, so compare
			// the spans as well: a label pointing at the primary location
			// explains nothing.
			if d.Related[0].Span == d.Primary {
				t.Errorf("the related location is the primary location, %q", src.TextOf(d.Primary))
			}
			if d.Related[0].Message == "" {
				t.Error("the related location has no explanation")
			}
		})
	}
}

// TestModesChangeSeverityAndNothingElse checks the property that makes the
// policy layer worth having. A mode decides whether a program is rejected; it
// must not decide what the problem is or where it is.
func TestModesChangeSeverityAndNothingElse(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		strict diagnostic.Severity
		// lenient is the severity outside strict mode. Some mistakes the
		// language tolerates there; most it does not.
		lenient diagnostic.Severity
	}{
		{"always an error", `let x: int = "hello"`, diagnostic.SeverityError, diagnostic.SeverityError},
		{"tolerated condition", `print(if 1 then { 2 } else { 3 })`, diagnostic.SeverityError, diagnostic.SeverityWarning},
		{"tolerated comparison", `print(1 == "wrong")`, diagnostic.SeverityError, diagnostic.SeverityWarning},
	} {
		t.Run(test.name, func(t *testing.T) {
			strictChecker, src := check(t, test.source, strict())
			strictDiagnostic := only(t, strictChecker)

			for _, mode := range []struct {
				name string
				mode types.TypeCheckingMode
				want diagnostic.Severity
			}{
				{"strict", strict(), test.strict},
				{"lenient", types.TypeCheckingMode{Enabled: true}, test.lenient},
				{"warn-only", types.TypeCheckingMode{Enabled: true, Strict: true, WarnOnly: true}, diagnostic.SeverityWarning},
			} {
				t.Run(mode.name, func(t *testing.T) {
					checker, _ := check(t, test.source, mode.mode)
					d := only(t, checker)

					if d.Severity != mode.want {
						t.Errorf("severity = %v, want %v", d.Severity, mode.want)
					}
					if d.Code != strictDiagnostic.Code {
						t.Errorf("code = %q, want %q; a mode must not change what the problem is",
							d.Code, strictDiagnostic.Code)
					}
					if d.Primary != strictDiagnostic.Primary {
						t.Errorf("points at %q, want %q; a mode must not change where the problem is",
							src.TextOf(d.Primary), src.TextOf(strictDiagnostic.Primary))
					}
					if (d.Severity == diagnostic.SeverityError) != checker.HasErrors() {
						t.Error("HasErrors disagrees with the severity of the only diagnostic")
					}
				})
			}
		})
	}
}

func TestDisabledCheckerReportsNothing(t *testing.T) {
	checker, _ := check(t, `let x: int = "hello" print(missing)`, types.TypeCheckingMode{})

	if len(checker.Diagnostics()) != 0 || checker.HasErrors() || checker.HasWarnings() {
		t.Errorf("a disabled checker reported %v", checker.Diagnostics())
	}
}

func TestDiagnosticsAreOrderedBySourcePosition(t *testing.T) {
	checker, src := check(t, "let a: int = \"first\"\nlet b: string = 2\nprint(missing)\n", strict())

	all := checker.Diagnostics()
	if len(all) != 3 {
		t.Fatalf("got %d diagnostics, want 3: %v", len(all), checker.GetErrors())
	}
	for i, want := range []string{`"first"`, "2", "missing"} {
		if got := src.TextOf(all[i].Primary); got != want {
			t.Errorf("diagnostic %d points at %q, want %q", i, got, want)
		}
	}
}

func TestStringAccessorsCarryPositionsWhenKnown(t *testing.T) {
	checker, _ := check(t, "let a = 1\nlet x: int = \"hello\"", strict())

	errors := checker.GetErrors()
	if len(errors) != 1 {
		t.Fatalf("got %v, want one error", errors)
	}
	if !strings.HasPrefix(errors[0], "check.flux:2:14: ") {
		t.Errorf("error = %q, want it to start with the position of the value", errors[0])
	}

	// A checker built without a source still reports, just without a location.
	result := parser.ParseSource(source.New(1, "nowhere.flux", `let x: int = "hello"`))
	unlocated := types.NewTypeCheckerWithConfig(strict())
	unlocated.CheckProgram(result.Program)
	messages := unlocated.GetErrors()
	if len(messages) != 1 || strings.Contains(messages[0], ":") == false {
		t.Fatalf("got %v, want one unlocated message", messages)
	}
	if unlocated.Diagnostics()[0].HasLocation() {
		t.Error("a checker with no source produced a located diagnostic")
	}
}

func TestLenientNotesExplainTheTolerance(t *testing.T) {
	checker, _ := check(t, `print(if 1 then { 2 } else { 3 })`, types.TypeCheckingMode{Enabled: true})
	d := only(t, checker)

	if len(d.Notes) == 0 {
		t.Fatal("a tolerated mistake does not say what the checker did instead")
	}
	if !strings.Contains(strings.Join(checker.GetWarnings(), "\n"), "treating as truthy") {
		t.Errorf("warnings = %v, want the note rendered", checker.GetWarnings())
	}
}

func TestCheckerCodesAreRegisteredInTheRightGroups(t *testing.T) {
	for code, want := range map[diagnostic.Code]diagnostic.Group{
		types.CodeAnnotationMismatch: diagnostic.GroupType,
		types.CodeArgumentType:       diagnostic.GroupType,
		types.CodeNotCallable:        diagnostic.GroupType,
		// A name that nothing declares is a mistake about bindings, not types.
		types.CodeUndefinedVariable:  diagnostic.GroupBinding,
		types.CodeDuplicateParameter: diagnostic.GroupBinding,
	} {
		if _, ok := diagnostic.Describe(code); !ok {
			t.Errorf("%s is not registered", code)
		}
		if group, ok := code.Group(); !ok || group != want {
			t.Errorf("%s is in group %q, want %q", code, group, want)
		}
	}
}

func TestBuiltinsAndAnnotatedFunctionsReportWithoutParameterProvenance(t *testing.T) {
	// print is built in and a function type written as an annotation has no
	// parameter names, so neither can offer a related location. The diagnostic
	// must still be reported, without a dangling label.
	for _, text := range []string{
		`let f: fn(int) -> int = fn(a: int) => a let g = f print(g("wrong"))`,
	} {
		checker, src := check(t, text, strict())
		d := only(t, checker)
		if d.Code != types.CodeArgumentType {
			t.Fatalf("code = %q, want %q", d.Code, types.CodeArgumentType)
		}
		if got := src.TextOf(d.Primary); got != `"wrong"` {
			t.Errorf("points at %q, want %q", got, `"wrong"`)
		}
		for _, related := range d.Related {
			if !related.Span.IsValid() {
				t.Error("a related location with no span was attached")
			}
		}
	}
}
