package parser_test

import (
	"github.com/pranavms13/flux-lang/parser"
	"testing"
)

func TestPhase3SyntaxBoundaries(t *testing.T) {
	for _, text := range []string{`print(-9223372036854775808)`, `print(-0009223372036854775808)`, `print(1-2)`, `print(1<=2 && 3!=4 || false)`, `print((fn(x)=>x)(1))`, `print((if true then [1] else [2])[0])`, `print("!") print("-") print("if")`, `let f=fn(x)=>x;f;(1)`, `print({let x=1;x;})`, `print({})`} {
		t.Run(text, func(t *testing.T) { mustParse(t, "syntax.flux", text) })
	}
	for _, test := range []struct{ text, at string }{
		{`9223372036854775808`, `9223372036854775808`},
		{`-9223372036854775809`, `9223372036854775809`},
		{`-(9223372036854775808)`, `9223372036854775808`},
		{`9999999999999999999999999999999999`, `9999999999999999999999999999999999`},
	} {
		result := parse(t, "range.flux", test.text)
		if !result.Failed() || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != parser.CodeIntegerRange || result.Source.TextOf(result.Diagnostics[0].Primary) != test.at {
			t.Fatalf("%s: %v", test.text, result.Diagnostics)
		}
	}
	for _, text := range []string{`1<2<3`, `1<=2>=3`, `true & false`, `true | false`, `print(1+)`, `print(!)`, `{let x=1:2}`, `print(1);;print(2)`} {
		if result := parse(t, "invalid.flux", text); !result.Failed() {
			t.Errorf("accepted %s", text)
		}
	}
}
func TestSemicolonPositionAndTrivia(t *testing.T) {
	result := mustParse(t, "separator.flux", "1 /* comment */ ;\n(2)")
	if len(result.Program.Statements) != 2 {
		t.Fatal("semicolon failed to separate expressions")
	}
	if got := spanText(t, result, result.Program.Statements[0].Separator); got != ";" {
		t.Fatalf("separator span=%q", got)
	}
	found := false
	for _, token := range result.Tokens() {
		if token.Value == "/* comment */" {
			found = true
		}
	}
	if !found {
		t.Fatal("comment trivia lost")
	}
}
