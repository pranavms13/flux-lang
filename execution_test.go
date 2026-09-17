package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/internal/testutil"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
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
		{"empty typed collections", `let xs: [int] = [] let d: {string: int} = {} print(xs) print(d)`, "[]\nmap[]\n"},
		{"collection equality", `print([1, 2] == [1, 2]) print({"a": 1} == {"a": 2})`, "true\nfalse\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { assertExecution(t, tt.source, tt.want) })
	}
}

func assertExecution(t *testing.T, source, want string) {
	t.Helper()
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	tc := types.NewTypeChecker()
	tc.CheckProgram(prog)
	if tc.HasErrors() {
		t.Fatalf("type errors: %v", tc.GetErrors())
	}
	for _, backend := range []string{"interpreter", "vm"} {
		t.Run(backend, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("execution panicked: %v", err)
				}
			}()
			got := testutil.CaptureOutput(t, func() {
				if backend == "interpreter" {
					runtime.Run(prog)
				} else {
					vm.New(compiler.NewFluxCompiler().Compile(prog)).Run()
				}
			})
			if got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
	}
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

func TestRuntimeFailures(t *testing.T) {
	for _, source := range []string{
		`let f = fn(x) => x f()`,
		`let f = fn(x) => x f(1, 2)`,
		`print()`,
		`print(1, 2)`,
		`let xs = [1] print(xs[2])`,
		`let xs = [1] print(xs[0 - 1])`,
		`let xs = [1] print(xs["wrong"])`,
		`let d = {"a": 1} print(d["missing"])`,
		`print(1 + "wrong")`,
		`print(missing)`,
		`let f = "print" f(1)`,
	} {
		prog, err := parser.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, backend := range []string{"interpreter", "vm"} {
			t.Run(backend+"/"+source, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Error("invalid program executed without an error")
					}
				}()
				if backend == "interpreter" {
					runtime.Run(prog)
				} else {
					vm.New(compiler.NewFluxCompiler().Compile(prog)).Run()
				}
			})
		}
	}
}
