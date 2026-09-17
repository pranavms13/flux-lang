package main

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

// These benchmarks exist to record a baseline before Phase 1 changes the
// parser, the checker, and both execution paths. Their numbers are a reference
// point, not a budget: a budget set before anyone knows the cost of carrying
// source positions would either be met by accident or missed for good reasons.
//
// The three shapes stress different parts of the pipeline. "small" is the
// script a user actually writes. "nested" is deeply recursive, so it measures
// the recursive descent in the parser, checker, and compiler rather than the
// per-statement cost. "large" is wide and flat, so it measures the per-statement
// cost and the environments that grow with it.

// nestedDepth is kept low because parsing nested conditionals costs exponential
// time; see BenchmarkParseNesting. A depth that looks unremarkable in source
// makes the benchmark, and the suite, unusable.
const nestedDepth = 6

// benchPrograms are built once so that the source text itself is not timed.
var benchPrograms = map[string]string{
	"small":  smallProgram(),
	"nested": nestedProgram(nestedDepth),
	"large":  largeProgram(300),
}

// benchOrder keeps the sub-benchmarks in a stable, readable order.
var benchOrder = []string{"small", "nested", "large"}

// compiledChunk keeps the compiler's result reachable so the loop that produces
// it cannot be optimized away. The module targets Go 1.23, which has neither
// b.Loop nor testing.B.KeepAlive.
var compiledChunk *vm.Chunk

func smallProgram() string {
	return `let add = fn(a: int, b: int): int => a + b
let greet = fn(name: string): string => "Hello, " + name
let xs = [1, 2, 3]
let person = {"name": "Alice", "city": "Paris"}
let total = add(xs[0], xs[2])
print(greet(person["name"]))
print(if total > 3 then { total } else { 0 })
`
}

// nestedProgram nests conditionals inside one function and returns a closure
// from another, so the recursive descent in every stage is exercised by a
// program that still type checks.
func nestedProgram(depth int) string {
	var source strings.Builder
	source.WriteString("let deep = fn(x: int): int => ")
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&source, "if x > %d then { ", i)
	}
	source.WriteString("x")
	for i := depth - 1; i >= 0; i-- {
		fmt.Fprintf(&source, " } else { %d }", i)
	}
	source.WriteString("\nlet make = fn(a: int): fn(int) -> int => fn(b: int): int => a + b\n")
	source.WriteString("print(make(deep(3))(4))\n")
	return source.String()
}

// largeProgram declares many bindings and a wide list, mirroring the shape that
// TestLargePrograms already covers for correctness.
func largeProgram(count int) string {
	var source strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&source, "let x%d = %d\n", i, i)
	}
	source.WriteString("let xs = [")
	for i := 0; i < count; i++ {
		if i > 0 {
			source.WriteString(", ")
		}
		fmt.Fprintf(&source, "x%d", i)
	}
	fmt.Fprintf(&source, "]\nprint(xs[%d])\n", count-1)
	return source.String()
}

// BenchmarkParseNesting records how parsing scales with expression nesting.
// It is separate because the answer is not linear: with MaxLookahead the parser
// re-tries each alternative of Expr over the whole nested subtree, so each added
// conditional multiplies the work. At the baseline commit a 450-byte program of
// sixteen nested conditionals takes seconds to parse. Grammar work in Phase 1
// and Phase 3 should watch this curve.
func BenchmarkParseNesting(b *testing.B) {
	for _, depth := range []int{2, 4, 6, 8} {
		source := nestedProgram(depth)
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := parser.Parse(source); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	forEachProgram(b, func(b *testing.B, source string) {
		b.SetBytes(int64(len(source)))
		for i := 0; i < b.N; i++ {
			if _, err := parser.Parse(source); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTypeCheck(b *testing.B) {
	forEachProgram(b, func(b *testing.B, source string) {
		prog := mustParse(b, source)
		for i := 0; i < b.N; i++ {
			checker := types.NewTypeChecker()
			checker.CheckProgram(prog)
			if checker.HasErrors() {
				b.Fatalf("benchmark program has type errors: %v", checker.GetErrors())
			}
		}
	})
}

func BenchmarkCompile(b *testing.B) {
	forEachProgram(b, func(b *testing.B, source string) {
		prog := mustParse(b, source)
		for i := 0; i < b.N; i++ {
			compiledChunk = compiler.NewFluxCompiler().Compile(prog)
		}
	})
}

func BenchmarkInterpret(b *testing.B) {
	forEachProgram(b, func(b *testing.B, source string) {
		prog := mustParse(b, source)
		options := runtime.Options{Output: io.Discard}
		for i := 0; i < b.N; i++ {
			if err := runtime.Run(prog, options); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkVM(b *testing.B) {
	forEachProgram(b, func(b *testing.B, source string) {
		chunk := compiler.NewFluxCompiler().Compile(mustParse(b, source))
		for i := 0; i < b.N; i++ {
			if err := vm.NewWithOutput(chunk, io.Discard).Run(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func forEachProgram(b *testing.B, run func(b *testing.B, source string)) {
	b.Helper()
	for _, name := range benchOrder {
		b.Run(name, func(b *testing.B) { run(b, benchPrograms[name]) })
	}
}

func mustParse(b *testing.B, source string) *ast.Program {
	b.Helper()
	prog, err := parser.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	return prog
}

// TestBenchmarkProgramsAreValid keeps the baseline numbers meaningful. A
// benchmark program that fails to type check would measure how fast the checker
// gives up, and one that disagrees between backends would measure two different
// computations.
func TestBenchmarkProgramsAreValid(t *testing.T) {
	for _, name := range benchOrder {
		t.Run(name, func(t *testing.T) {
			text := benchPrograms[name]
			result := parser.ParseSource(source.New(1, name+".flux", text))
			if result.Failed() {
				t.Fatal(result.Diagnostics)
			}
			checker := types.NewTypeCheckerForSource(result.Source, types.TypeCheckingMode{Enabled: true})
			checker.CheckProgram(result.Program)
			if checker.HasErrors() {
				t.Fatalf("type errors: %v", checker.GetErrors())
			}
			interpreted, err := backends[0].run(result)
			if err != nil {
				t.Fatalf("interpreter: %v", err)
			}
			executed, err := backends[1].run(result)
			if err != nil {
				t.Fatalf("vm: %v", err)
			}
			if interpreted != executed {
				t.Errorf("backends disagree: interpreter %q, vm %q", interpreted, executed)
			}
			if interpreted == "" {
				t.Error("the program produced no output, so execution is not being measured")
			}
		})
	}
}
