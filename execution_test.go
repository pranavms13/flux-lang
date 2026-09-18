package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

func TestExecutionParity(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"print in block", `let x = if true then { print("inside") 42 } else { 0 } print(x)`, "inside\n42\n"},
		{"print in function", `let log = fn(x: int): void => print(x) log(7)`, "7\n"},
		{"print returns void", `let x: void = print(1)`, "1\n"},
		{"booleans", `print(true) print(false) print(yes) print(no)`, "true\nfalse\ntrue\nfalse\n"},
		{"false branch", `print(if false then { 1 } else { 2 })`, "2\n"},
		{"arithmetic", `print(10 - 3 - 2) print(1 + 2 > 2) print(1 < 2 + 3)`, "5\ntrue\ntrue\n"},
		{"grouping", `print((10 - 3) - 2)`, "5\n"},
		{"escaped string", `print("say \"hi\"")`, "say \"hi\"\n"},
		{"conditional argument", `let add = fn(a: int, b: int): int => a + b print(add(10, if true then { 2 } else { 3 }))`, "12\n"},
		{"block argument", `let add = fn(a: int, b: int): int => a + b print(add(10, { 1 2 }))`, "12\n"},
		{"block list", `let xs = [{ 1 2 }, { 3 4 }] print(xs[0]) print(xs[1])`, "2\n4\n"},
		{"closure", `let make = fn(x: int) => fn(y: int): int => x + y let add = make(10) print(add(3))`, "13\n"},
		{"dict duplicate", `let d = {"a": 1, "a": 2} print(d["a"])`, "2\n"},
		{"dict evaluation order", `let key = fn(): string => { print("key") "a" } let value = fn(): int => { print("value") 1 } let d = {key(): value()} print(d["a"])`, "key\nvalue\n1\n"},
		{"empty typed collections", `let xs: [int] = [] let d: {string: int} = {} print(xs) print(d)`, "[]\n{}\n"},
		{"collection equality", `print([1, 2] == [1, 2]) print({"a": 1} == {"a": 2})`, "true\nfalse\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { assertExecution(t, tt.source, tt.want) })
	}
}

// assertExecution parses and checks a program, then verifies matching
// successful output from both execution backends.
func assertExecution(t *testing.T, text, want string) {
	t.Helper()
	result := parser.ParseSource(source.New(1, "parity.flux", text))
	if result.Failed() {
		t.Fatalf("parse: %v", result.Diagnostics)
	}
	checker := types.NewTypeCheckerForSource(result.Source, types.TypeCheckingMode{Enabled: true})
	checker.CheckProgram(result.Program)
	if checker.HasErrors() {
		t.Fatalf("type errors: %v", checker.GetErrors())
	}
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			got, err := backend.run(result)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
	}
}

// backends runs the same program through both engines. Output is captured
// through an injected writer, so these tests no longer depend on replacing
// os.Stdout and can run alongside anything else.
var backends = []struct {
	name string
	run  func(result parser.Result) (string, error)
}{
	{"interpreter", func(result parser.Result) (string, error) {
		var out bytes.Buffer
		err := runtime.Run(result.Program, runtime.Options{Output: &out, Source: result.Source})
		return out.String(), err
	}},
	{"vm", func(result parser.Result) (string, error) {
		var out bytes.Buffer
		chunk := compiler.NewFluxCompilerForSource(result.Source).Compile(result.Program)
		err := vm.NewWithOutput(chunk, &out).Run()
		return out.String(), err
	}},
}

func TestLargePrograms(t *testing.T) {
	var source, want strings.Builder
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&source, "let x%d = %d\n", i, i)
	}
	source.WriteString("print(if true then { x299 } else { 0 })")
	assertExecution(t, source.String(), "299\n")
	source.Reset()
	source.WriteString("let xs = [")
	for i := 0; i < 300; i++ {
		if i > 0 {
			source.WriteString(",")
		}
		fmt.Fprint(&source, i)
	}
	source.WriteString("] print(xs[299])")
	assertExecution(t, source.String(), "299\n")
	source.Reset()
	source.WriteString("let x = if false then {")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&source, "print(%d) ", i)
	}
	source.WriteString("1 } else { 2 } print(x)")
	want.WriteString("2\n")
	assertExecution(t, source.String(), want.String())
}

// The examples under examples/ are exercised by TestFixturePrograms, which runs
// each of them in every type-checking mode against the outcome declared in
// internal/fixtures. It replaced a table here that listed four example names and
// their expected output, and could not say what the other examples were for.

// TestRuntimeFailures checks the Phase 1 criterion that both backends agree on
// what went wrong and where, not merely that both refused to finish.
func TestRuntimeFailures(t *testing.T) {
	for _, test := range []struct {
		source string
		code   diagnostic.Code
		at     string
	}{
		{`let f = fn(x) => x f()`, fault.CodeArgumentCount, "()"},
		{`let f = fn(x) => x f(1, 2)`, fault.CodeArgumentCount, "(1, 2)"},
		{`print()`, fault.CodeArgumentCount, "()"},
		{`print(1, 2)`, fault.CodeArgumentCount, "(1, 2)"},
		{`let xs = [1] print(xs[2])`, fault.CodeIndexRange, "[2]"},
		{`let xs = [1] print(xs[0 - 1])`, fault.CodeIndexRange, "[0 - 1]"},
		{`let xs = [1] print(xs["wrong"])`, fault.CodeIndexType, `["wrong"]`},
		{`let d = {"a": 1} print(d["missing"])`, fault.CodeMissingKey, `["missing"]`},
		{`print(1 + "wrong")`, fault.CodeOperandType, `+ "wrong"`},
		{`print(missing)`, types.CodeUndefinedVariable, "missing"},
		{`let f = "print" f(1)`, fault.CodeNotCallable, "(1)"},
	} {
		t.Run(test.source, func(t *testing.T) {
			result := parser.ParseSource(source.New(7, "failure.flux", test.source))
			if result.Failed() {
				t.Fatalf("parse: %v", result.Diagnostics)
			}
			reported := map[string]*fault.Error{}
			for _, backend := range backends {
				_, err := backend.run(result)
				if err == nil {
					t.Fatalf("%s ran an invalid program without an error", backend.name)
				}
				failure, ok := err.(*fault.Error)
				if !ok {
					t.Fatalf("%s returned %T, want a *fault.Error", backend.name, err)
				}
				if failure.Code != test.code {
					t.Errorf("%s reported %q, want %q", backend.name, failure.Code, test.code)
				}
				if got := result.Source.Text()[failure.Where.Start:failure.Where.End]; got != test.at {
					t.Errorf("%s points at %q, want %q", backend.name, got, test.at)
				}
				if got := result.Source.TextOf(failure.Diagnostic().Primary); got != test.at {
					t.Errorf("%s diagnostic points at %q, want %q in the original source", backend.name, got, test.at)
				}
				reported[backend.name] = failure
			}
			interpreted, executed := reported["interpreter"], reported["vm"]
			if interpreted.Message != executed.Message {
				t.Errorf("backends disagree on the message: %q and %q",
					interpreted.Message, executed.Message)
			}
			if interpreted.Where != executed.Where {
				t.Errorf("backends disagree on the location: %v and %v",
					interpreted.Where, executed.Where)
			}
		})
	}
}

// TestRuntimeFailureTraces checks that a failure inside a function says which
// call led there.
func TestRuntimeFailureTraces(t *testing.T) {
	const text = "let inner = fn(x) => x + \"no\"\nlet outer = fn(y) => inner(y)\nprint(outer(1))\n"
	result := parser.ParseSource(source.New(1, "trace.flux", text))
	if result.Failed() {
		t.Fatalf("parse: %v", result.Diagnostics)
	}
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			_, err := backend.run(result)
			failure, ok := err.(*fault.Error)
			if !ok {
				t.Fatalf("got %v (%T), want a *fault.Error", err, err)
			}
			if failure.Code != fault.CodeOperandType {
				t.Fatalf("code = %q, want %q", failure.Code, fault.CodeOperandType)
			}
			if failure.Where.Line != 1 {
				t.Errorf("failure reported on line %d, want line 1", failure.Where.Line)
			}
			if len(failure.Trace) != 2 {
				t.Fatalf("trace has %d frames, want 2: %+v", len(failure.Trace), failure.Trace)
			}
			// Innermost first: the call to inner, then the call to outer.
			if got, want := failure.Trace[0].Function, "inner"; got != want {
				t.Errorf("innermost frame is %q, want %q", got, want)
			}
			if got, want := failure.Trace[0].Call.Line, 2; got != want {
				t.Errorf("innermost call site is on line %d, want %d", got, want)
			}
			if got, want := failure.Trace[1].Function, "outer"; got != want {
				t.Errorf("outer frame is %q, want %q", got, want)
			}
			if got, want := failure.Trace[1].Call.Line, 3; got != want {
				t.Errorf("outer call site is on line %d, want %d", got, want)
			}
		})
	}
}
