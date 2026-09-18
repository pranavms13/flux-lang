package types_test

import (
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/types"
)

// TestTypeCheckingModes checks rejection, warning, and disabled behavior for
// representative programs under each checking mode.
func TestTypeCheckingModes(t *testing.T) {
	cases := []struct {
		name, source, diagnostic string
		lenientWarning           bool
	}{
		{"assignment", `let x: int = "hello"`, "type mismatch", false},
		{"no implicit conversion", `let x: string = 1`, "type mismatch", false},
		{"call argument", `let f = fn(x: int) => x print(f("wrong"))`, "argument 1", false},
		{"return", `let f = fn(): int => "wrong"`, "return type mismatch", false},
		{"addition", `print(1 + "wrong")`, "invalid operands", false},
		{"subtraction", `print("wrong" - 1)`, "invalid operands", false},
		{"comparison", `print("wrong" > 1)`, "invalid operands", false},
		{"known invalid operand", `let f = fn(x) => true + x`, "invalid operands", false},
		{"condition", `print(if 1 then { 2 } else { 3 })`, "if condition must be bool", true},
		{"branches", `let x = if true then { 2 } else { "wrong" }`, "if branches must have same type", true},
		{"equality", `print(1 == "wrong")`, "cannot compare different types", true},
		{"list", `let x = [1, "wrong"]`, "list element", false},
		{"dictionary", `let x = {"a": 1, "b": "wrong"}`, "dictionary value", false},
		{"dictionary key", `let x = {[1]: 2}`, "dictionary key must be", false},
		{"list index", `let x = [1] print(x["wrong"])`, "list index must be int", false},
		{"dict index", `let x = {"a": 1} print(x[2])`, "dictionary key must be string", false},
		{"arity", `let f = fn(x) => x f()`, "function expects", false},
		{"duplicate parameters", `let f = fn(x, x) => x`, "duplicate parameter", false},
		{"undefined", `print(missing)`, "undefined variable", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []struct {
				name   string
				config types.TypeCheckingMode
			}{
				{"strict", types.TypeCheckingMode{Enabled: true, Strict: true}},
				{"lenient", types.TypeCheckingMode{Enabled: true}},
				{"warn-only", types.TypeCheckingMode{Enabled: true, Strict: true, WarnOnly: true}},
				{"disabled", types.TypeCheckingMode{}},
			} {
				t.Run(mode.name, func(t *testing.T) {
					tc := types.NewTypeCheckerWithConfig(mode.config)
					tc.CheckProgram(prog)
					if !mode.config.Enabled {
						if tc.HasErrors() || tc.HasWarnings() {
							t.Fatal("disabled checker emitted diagnostics")
						}
						return
					}
					warning := mode.config.WarnOnly || (!mode.config.Strict && tt.lenientWarning)
					messages := tc.GetErrors()
					if warning {
						if tc.HasErrors() {
							t.Fatalf("unexpected errors: %v", tc.GetErrors())
						}
						messages = tc.GetWarnings()
					}
					if !strings.Contains(strings.Join(messages, "\n"), tt.diagnostic) {
						t.Fatalf("diagnostics = %v, want %q", messages, tt.diagnostic)
					}
				})
			}
		})
	}
}

func TestInferenceAndTypeConversions(t *testing.T) {
	for _, source := range []string{
		`let x: int = 1 + 2`,
		`let f = fn(x) => x + x print(f(2))`,
		`let f = fn(xs) => xs[0] print(f([2]))`,
		`let apply = fn(f, x) => f(x) let id = fn(x) => x print(apply(id, 2))`,
		`let xs: [int] = [] let d: {string: int} = {}`,
		`let f: fn(int) -> int = fn(x) => x`,
	} {
		prog, err := parser.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		tc := types.NewTypeCheckerWithConfig(types.TypeCheckingMode{Enabled: true, Strict: true})
		tc.CheckProgram(prog)
		if tc.HasErrors() {
			t.Errorf("%s: %v", source, tc.GetErrors())
		}
	}
	for _, typ := range []types.FluxType{
		types.IntType{}, types.StringType{}, types.BoolType{}, types.VoidType{},
		types.ListType{ElementType: types.IntType{}},
		types.DictType{KeyType: types.StringType{}, ValueType: types.BoolType{}},
		types.FunctionType{ParamTypes: []types.FluxType{types.IntType{}}, ReturnType: types.StringType{}},
	} {
		node, err := types.ConvertFluxTypeToAST(typ)
		if err != nil {
			t.Fatal(err)
		}
		converted, err := types.ConvertASTType(node)
		if err != nil || !types.TypesEqual(typ, converted) {
			t.Fatalf("round trip %s: %v, %v", typ, converted, err)
		}
	}
	a := types.FunctionType{ParamTypes: []types.FluxType{types.UnknownType{}, types.IntType{}}, ReturnType: types.VoidType{}}
	b := types.FunctionType{ParamTypes: []types.FluxType{types.IntType{}, types.UnknownType{}}, ReturnType: types.VoidType{}}
	if !types.TypesEqual(a, b) || !types.TypesEqual(b, a) {
		t.Fatal("nested unknown types should be compatible symmetrically")
	}
}
