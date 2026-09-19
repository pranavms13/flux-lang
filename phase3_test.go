package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/internal/fixtures"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/render"
	"github.com/pranavms13/flux-lang/resolver"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/value"
	"github.com/pranavms13/flux-lang/vm"
)

func TestPhase3ExpressionsAndClosures(t *testing.T) {
	for _, test := range []struct{ name, text, want string }{
		{"precedence", `print(1+2*3) print(10-3-2) print(-2*3) print(!(1<2)) print(1<2 == true && false || 3>=3) print(8/2*3%5) print(2!=3) print(3<=3)`, "7\n5\n-6\nfalse\ntrue\n2\ntrue\ntrue\n"},
		{"primary expressions", `print((fn(x: int): int => x+1)(3)) print((if true then [4] else [5])[0])`, "4\n4\n"},
		{"operator strings", `let x="!" print(x) print("-"+"+") print("if") print("(") print(";") print("==")`, "!\n-+\nif\n(\n;\n==\n"},
		{"short circuit", `print(false && (1/0>0)) print(true || (1/0>0)) print(false && {print("bad") true}) print(true || {print("bad") false}) let side=fn(): bool=>{print("bad");true};print(false && side()) print(true || side())`, "false\ntrue\nfalse\ntrue\nfalse\ntrue\n"},
		{"nested stack", `let f=fn(a: bool,b: bool): [bool] => [a,b] print(f(true && false || true,false || true && false)) print([true || false,false && true]) print({"k": true && (false || true)}["k"]) print(10+{false && true; 2})`, "[true, false]\n[true, false]\ntrue\n12\n"},
		{"numeric limits", `print(9223372036854775807) print(-9223372036854775808) print(-0009223372036854775808) print(-7/2) print(-7%2) print(7%-2) print(-9223372036854775808%-1) print(--2)`, "9223372036854775807\n-9223372036854775808\n-9223372036854775808\n-3\n-1\n1\n0\n2\n"},
		{"shadowing", `let x=1; let f={let before=fn(): int=>x; let x=2; let after=fn(): int=>x; fn(): [int]=>[before(),after()]};print(f()) print(x)`, "[1, 2]\n1\n"},
		{"returned captures", `let make=fn(x: int)=>{let y=x*2;fn(z: int)=>fn(): int=>x+y+z};let a=make(3)(4);let b=make(5)(1);print(a()) print(b())`, "13\n16\n"},
		{"block void", `print({let x=1;}) print({1;let x=2}) print({})`, "<void>\n<void>\n{}\n"},
		{"typed recursion", `let factorial=fn(n: int): int=>{let base=n<=1;if base then 1 else n*factorial(n-1)};print(factorial(5))`, "120\n"},
		{"annotated variable recursion", `let sum: fn(int)->int=(fn(n)=>if n<=0 then 0 else n+sum(n-1));print(sum(4))`, "10\n"},
		{"local recursion", `let make=fn(offset: int)=>{let sum=fn(n: int): int=>if n==0 then offset else n+sum(n-1);sum};let a=make(10);let b=make(20);print(a(3)) print(b(3))`, "16\n26\n"},
		{"builtin shadow", `let original=print;let print=fn(x: int): int=>x+1;original(print(2))`, "3\n"},
		{"display", `print(fn(x)=>x) print(print) print([fn()=>1]) print({"z": ["x", "y"], "a": []}) print(print("x"))`, "<function>\n<function>\n[<function>]\n{\"a\": [], \"z\": [\"x\", \"y\"]}\nx\n<void>\n"},
		{"equality", `print([{"x":1},{"x":2}] == [{"x":1},{"x":2}]) print({"a":1,"b":2} != {"b":2,"a":1}) print(print(1)==print(2))`, "true\nfalse\n1\n2\ntrue\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed := parser.ParseSource(source.New(1, "phase3.flux", test.text))
			if parsed.Failed() {
				t.Fatal(parsed.Diagnostics)
			}
			for _, mode := range fixtures.Modes {
				checker := types.NewTypeCheckerForSource(parsed.Source, mode.TypeChecking())
				checker.CheckProgram(parsed.Program)
				if checker.HasErrors() {
					t.Fatalf("%s: %v", mode, checker.Diagnostics())
				}
				for _, backend := range backends {
					got, err := backend.run(parsed)
					if err != nil || got != test.want {
						t.Fatalf("%s/%s: %q, %v; want %q", mode, backend.name, got, err, test.want)
					}
				}
			}
			// Nested capture metadata and int64 constants must survive serialization.
			encoded, err := vm.Encode(compiler.NewFluxCompilerForSource(parsed.Source).Compile(parsed.Program), nil)
			if err != nil {
				t.Fatal(err)
			}
			program, err := vm.Decode(encoded)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := vm.NewWithOutput(program.Chunk, &out).Run(); err != nil || out.String() != test.want {
				t.Fatalf("serialized VM: %q, %v", out.String(), err)
			}
		})
	}
}

func TestPhase3RuntimeFailures(t *testing.T) {
	for _, test := range []struct {
		text string
		code diagnostic.Code
		at   string
	}{
		{`print(9223372036854775807+1)`, fault.CodeIntOverflow, "+1"},
		{`print(-9223372036854775808-1)`, fault.CodeIntOverflow, "-1"},
		{`print(9223372036854775807*2)`, fault.CodeIntOverflow, "*2"},
		{`print(-9223372036854775808/-1)`, fault.CodeIntOverflow, "/-1"},
		{`print(-(-9223372036854775808))`, fault.CodeIntOverflow, "-(-9223372036854775808)"},
		{`print(true && (1/0>0))`, fault.CodeZeroDivisor, "/0"},
		{`print(false || (1%0>0))`, fault.CodeZeroDivisor, "%0"},
		{`print(if 1 then 2 else 3)`, fault.CodeConditionType, "1"},
		{`print(1 && true)`, fault.CodeConditionType, "1"},
		{`print(false || 1)`, fault.CodeConditionType, "1"},
		{`print(true && 1)`, fault.CodeConditionType, "1"},
		{`print(!1)`, fault.CodeOperandType, "!1"},
		{`print(-true)`, fault.CodeOperandType, "-true"},
		{`let f=fn()=>1;print([0,f]==[1,f])`, fault.CodeIncomparable, "[0,f]==[1,f]"},
		{`print(print==print)`, fault.CodeIncomparable, "print==print"},
		{`print([1][9223372036854775807])`, fault.CodeIndexRange, "[9223372036854775807]"},
		{`print({[1]: 2})`, fault.CodeInvalidKey, "[1]"},
		{`print({"a":1}[[1]])`, fault.CodeInvalidKey, "[[1]]"},
	} {
		t.Run(test.text, func(t *testing.T) {
			parsed := parser.ParseSource(source.New(1, "failure.flux", test.text))
			if parsed.Failed() {
				t.Fatal(parsed.Diagnostics)
			}
			for _, backend := range backends {
				_, err := backend.run(parsed)
				e, ok := err.(*fault.Error)
				if !ok || e.Code != test.code {
					t.Fatalf("%s: %v, want %s", backend.name, err, test.code)
				}
				if got := parsed.Source.TextOf(e.Diagnostic().Primary); got != test.at {
					t.Errorf("%s points at %q, want %q", backend.name, got, test.at)
				}
			}
		})
	}
}

func TestPhase3StaticFailures(t *testing.T) {
	for _, test := range []struct {
		text    string
		code    diagnostic.Code
		at      string
		binding bool
	}{
		{`let x=x`, resolver.CodeSelfInitialization, "x", true},
		{`let x=(fn()=>x)()`, resolver.CodeSelfInitialization, "x", true},
		{`let x=1;{let x=x}`, resolver.CodeSelfInitialization, "x", true},
		{`let x=1;let x=2`, resolver.CodeDuplicateDeclaration, "let x=2", true},
		{`let f=fn(x,x)=>x`, resolver.CodeDuplicateParameter, "x", true},
		{`{let x=1};print(x)`, resolver.CodeUndefined, "x", true},
		{`let f=fn(): int=>g();let g=fn(): int=>f()`, resolver.CodeUndefined, "g", true},
		{`let f=fn(n)=>if n==0 then 0 else f(n-1)`, types.CodeRecursiveSignature, "let f=fn(n)=>if n==0 then 0 else f(n-1)", false},
		{`let f=fn(n): int=>if n==0 then 0 else f(n-1)`, types.CodeRecursiveSignature, "let f=fn(n): int=>if n==0 then 0 else f(n-1)", false},
		{`let f: fn(int)->int=fn(n)=>if n==0 then "bad" else f(n-1)`, types.CodeBranchMismatch, "f(n-1)", false},
		{`print(false && (1+true>0))`, types.CodeOperandType, "1+true", false},
		{`print(true || 1)`, types.CodeOperandType, "true || 1", false},
		{`let f=fn()=>1;print([f]==[f])`, types.CodeIncomparable, "[f]==[f]", false},
	} {
		t.Run(test.text, func(t *testing.T) {
			parsed := parser.ParseSource(source.New(1, "static.flux", test.text))
			if parsed.Failed() {
				t.Fatal(parsed.Diagnostics)
			}
			for _, mode := range fixtures.Modes {
				checker := types.NewTypeCheckerForSource(parsed.Source, mode.TypeChecking())
				checker.CheckProgram(parsed.Program)
				if !test.binding && mode == fixtures.ModeDisabled {
					if len(checker.Diagnostics()) != 0 {
						t.Fatal(checker.Diagnostics())
					}
					continue
				}
				ds := checker.Diagnostics()
				found := false
				for _, d := range ds {
					if d.Code == test.code && parsed.Source.TextOf(d.Primary) == test.at {
						found = true
					}
				}
				if !found {
					t.Fatalf("%s: want %s at %q, got %v", mode, test.code, test.at, ds)
				}
				if test.binding && !checker.HasErrors() {
					t.Fatalf("%s downgraded binding failure", mode)
				}
			}
		})
	}
}

func TestPhase3CallDepth(t *testing.T) {
	parsed := parser.ParseSource(source.New(1, "recursive.flux", `let forever=fn(n: int): int=>forever(n+1);forever(0)`))
	for _, limit := range []int{3, value.DefaultMaxDepth} {
		var output bytes.Buffer
		chunk := compiler.NewFluxCompilerForSource(parsed.Source).Compile(parsed.Program)
		machine := vm.NewWithOutput(chunk, &output)
		machine.MaxDepth = limit
		failures := []error{runtime.Run(parsed.Program, runtime.Options{Source: parsed.Source, Output: &output, MaxDepth: limit}), machine.Run()}
		for _, err := range failures {
			e, ok := err.(*fault.Error)
			if !ok || e.Code != fault.CodeCallDepth {
				t.Fatalf("got %v", err)
			}
			if len(e.Trace) != limit {
				t.Fatalf("trace length=%d, want %d", len(e.Trace), limit)
			}
			if got := parsed.Source.TextOf(e.Diagnostic().Primary); got != "(n+1)" {
				t.Fatalf("location %q", got)
			}
			rendered := (render.Renderer{Source: parsed.Source}).Failure(e)
			if got := strings.Count(rendered, "  in "); got != min(limit, fault.MaxTraceFrames) {
				t.Fatalf("rendered %d frames", got)
			}
		}
	}
}

func TestPhase3LargeLocalAndCaptureOperands(t *testing.T) {
	var text strings.Builder
	text.WriteString("let make=fn(): fn()->int=>{")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&text, "let x%d=%d;", i, i)
	}
	text.WriteString("fn(): int=>x0+x299};print(make()())")
	assertExecution(t, text.String(), "299\n")
}
