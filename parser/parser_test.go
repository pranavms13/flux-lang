package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/source"
)

// parse creates a named source snapshot and returns the complete parser result
// for assertions.
func parse(t *testing.T, name, text string) parser.Result {
	t.Helper()
	return parser.ParseSource(source.New(1, name, text))
}

// mustParse requires a successful parse and retains the source snapshot for
// span checks.
func mustParse(t *testing.T, name, text string) parser.Result {
	t.Helper()
	result := parse(t, name, text)
	if result.Failed() {
		t.Fatalf("parse %s: %v", name, result.Diagnostics)
	}
	return result
}

// spanText is what almost every assertion here reduces to: the source a node
// claims to cover must be the source it actually covers.
func spanText(t *testing.T, result parser.Result, node ast.Positioned) string {
	t.Helper()
	if !node.HasPosition() {
		t.Fatalf("%T has no recorded position", node)
	}
	return result.Source.TextOf(node.Span(result.Source.ID()))
}

// TestPositionsCoverEveryKindOfConstruct compares AST spans with the exact
// source text for each grammar construct.
func TestPositionsCoverEveryKindOfConstruct(t *testing.T) {
	const text = `let greet = fn(name: string): string => "Hello, " + name
let xs: [int] = [1, 2, 3]
print(greet("Flux"))
print(xs[1] > 2)
`
	result := mustParse(t, "positions.flux", text)
	program := result.Program
	if got, want := len(program.Statements), 4; got != want {
		t.Fatalf("parsed %d statements, want %d", got, want)
	}

	greet := program.Statements[0]
	xs := program.Statements[1]
	call := program.Statements[2]
	comparison := program.Statements[3]

	funcExpr := greet.Let.Expr.Func
	sum := greet.Let.Expr.Func.Body.Bin.Left
	greeting := primary(t, call.Expr).Postfix[0].Call
	indexed := primary(t, primary(t, comparison.Expr).Postfix[0].Call.Args[0])

	for _, test := range []struct {
		what string
		node ast.Positioned
		want string
	}{
		{"statement", greet, "let greet = fn(name: string): string => \"Hello, \" + name"},
		{"declaration", greet.Let, "let greet = fn(name: string): string => \"Hello, \" + name"},
		{"function literal", funcExpr, "fn(name: string): string => \"Hello, \" + name"},
		{"parameter", funcExpr.Params[0], "name: string"},
		{"parameter annotation", funcExpr.Params[0].TypeAnno, ": string"},
		{"return annotation", funcExpr.ReturnAnno, ": string"},
		{"annotated type", funcExpr.ReturnAnno.Type, "string"},
		{"string term", sum.Left.Base.Term, `"Hello, "`},
		// An operator's position comes from the node the operator begins, so
		// the addition covers the operator and its right operand.
		{"operator and operand", sum.Rest[0], "+ name"},
		{"identifier term", sum.Rest[0].Right.Base.Term, "name"},
		{"annotated declaration", xs.Let, "let xs: [int] = [1, 2, 3]"},
		{"list type annotation", xs.Let.TypeAnno, ": [int]"},
		{"list literal", primary(t, xs.Let.Expr).Base.List, "[1, 2, 3]"},
		{"list element", primary(t, xs.Let.Expr).Base.List.Elems[2], "3"},
		{"call arguments", greeting, `(greet("Flux"))`},
		{"nested call argument", greeting.Args[0], `greet("Flux")`},
		{"index", indexed.Postfix[0].Index, "[1]"},
		{"comparison", primary(t, comparison.Expr).Postfix[0].Call.Args[0].Bin.Rest[0], "> 2"},
	} {
		t.Run(test.what, func(t *testing.T) {
			if got := spanText(t, result, test.node); got != test.want {
				t.Errorf("span covers %q, want %q", got, test.want)
			}
		})
	}
}

// TestEndPositionSemantics pins behavior this package depends on but does not
// own. Participle sets EndPos from the next raw token, which makes it the
// exclusive end of the construct. A Participle upgrade that changed this to the
// last consumed token instead would shift every span by one token, silently.
func TestEndPositionSemantics(t *testing.T) {
	t.Run("a node ends at its last token, not at the trivia after it", func(t *testing.T) {
		const text = "let a = 1   // trailing comment\nlet b = 2"
		result := mustParse(t, "end.flux", text)
		if got, want := spanText(t, result, result.Program.Statements[0]), "let a = 1"; got != want {
			t.Errorf("span covers %q, want %q", got, want)
		}
	})

	t.Run("the end offset is exclusive", func(t *testing.T) {
		const text = "let a = 1"
		result := mustParse(t, "end.flux", text)
		span := result.Program.Statements[0].Span(result.Source.ID())
		if got, want := span.End, len(text); got != want {
			t.Errorf("end offset = %d, want %d (one past the last byte)", got, want)
		}
		if got, want := span.Start, 0; got != want {
			t.Errorf("start offset = %d, want %d", got, want)
		}
	})

	t.Run("a construct ending at end of input reaches it", func(t *testing.T) {
		const text = "print(1)"
		result := mustParse(t, "eof.flux", text)
		if got, want := spanText(t, result, result.Program.Statements[0]), "print(1)"; got != want {
			t.Errorf("span covers %q, want %q", got, want)
		}
	})

	t.Run("a node starts at its first token, not at the trivia before it", func(t *testing.T) {
		const text = "  // leading\n  let a = 1"
		result := mustParse(t, "lead.flux", text)
		if got, want := spanText(t, result, result.Program.Statements[0]), "let a = 1"; got != want {
			t.Errorf("span covers %q, want %q", got, want)
		}
	})
}

// TestPositionsSurviveUnicodeAndCRLF checks byte spans and displayed positions
// across multibyte characters and Windows line endings.
func TestPositionsSurviveUnicodeAndCRLF(t *testing.T) {
	// Identifiers are ASCII, so the non-ASCII text lives where Flux allows it:
	// in a string literal and in a comment. Both shift byte offsets away from
	// columns, which is the point.
	const text = "let a = \"\U0001F600\" print(a)\r\n" +
		"// a comment with \u00e9 and \U0001F600\r\n" +
		"let b = 2\r\n" +
		"print(b)\r\n"
	result := mustParse(t, "unicode.flux", text)

	if got, want := len(result.Program.Statements), 4; got != want {
		t.Fatalf("parsed %d statements, want %d", got, want)
	}

	// The emoji is four bytes and one rune, so the byte column and every other
	// column part ways at the statement that follows it on the same line.
	printA := result.Program.Statements[1]
	if got, want := spanText(t, result, printA), "print(a)"; got != want {
		t.Fatalf("span covers %q, want %q", got, want)
	}
	position := result.Source.Position(printA.Span(result.Source.ID()).Start)
	// Twelve characters precede it, so it is the thirteenth; the emoji
	// counts twice in UTF-16 and the byte column is further along still.
	want := source.Position{Line: 1, Byte: 16, Rune: 13, UTF16: 14, Display: 13}
	if position != want {
		t.Errorf("position = %+v, want %+v", position, want)
	}

	// A comment full of multi-byte text is elided, but its bytes still count.
	for index, wantLine := range map[int]int{2: 3, 3: 4} {
		statement := result.Program.Statements[index]
		if got := result.Source.Position(statement.Span(result.Source.ID()).Start).Line; got != wantLine {
			t.Errorf("statement %d starts on line %d, want %d", index, got, wantLine)
		}
	}

	// CRLF must not leave a carriage return inside a span.
	if got, want := spanText(t, result, result.Program.Statements[2]), "let b = 2"; got != want {
		t.Errorf("span covers %q, want %q", got, want)
	}
}

// TestLexicalFailureReportsOneCharacter checks that invalid characters receive
// a single-character span and syntax code.
func TestLexicalFailureReportsOneCharacter(t *testing.T) {
	for _, test := range []struct {
		name, text, want string
	}{
		{"ascii", "let a = 1\nlet b = #\n", "#"},
		{"multi-byte", "let é = 1\n", "é"},
		{"supplementary plane", "let a = 1\n\U0001F600\n", "\U0001F600"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := parse(t, "bad.flux", test.text)
			d := onlyDiagnostic(t, result)
			if d.Code != parser.CodeInvalidCharacter {
				t.Errorf("code = %q, want %q", d.Code, parser.CodeInvalidCharacter)
			}
			// The lexer reports the whole unmatched remainder; the diagnostic
			// must underline only the character that forms no token.
			if got := result.Source.TextOf(d.Primary); got != test.want {
				t.Errorf("span covers %q, want %q", got, test.want)
			}
			if !strings.Contains(d.Message, test.want) {
				t.Errorf("message %q does not name the offending character", d.Message)
			}
		})
	}
}

// TestSyntaxFailureLocatesTheOffendingToken verifies the token span and
// expected-token guidance for parse failures.
func TestSyntaxFailureLocatesTheOffendingToken(t *testing.T) {
	const text = "let a = 1\nlet b = let\n"
	result := parse(t, "syntax.flux", text)

	d := onlyDiagnostic(t, result)
	if d.Code != parser.CodeUnexpectedToken {
		t.Fatalf("code = %q, want %q", d.Code, parser.CodeUnexpectedToken)
	}
	if got, want := result.Source.TextOf(d.Primary), "let"; got != want {
		t.Errorf("span covers %q, want %q", got, want)
	}
	position := result.Source.Position(d.Primary.Start)
	if position.Line != 2 || position.Display != 9 {
		t.Errorf("position = line %d column %d, want line 2 column 9", position.Line, position.Display)
	}
	if !strings.Contains(d.Message, "expected") {
		t.Errorf("message %q does not say what was expected", d.Message)
	}
}

// TestUnexpectedEndOfInputIsItsOwnDiagnostic checks the dedicated EOF code and
// empty end-of-file span.
func TestUnexpectedEndOfInputIsItsOwnDiagnostic(t *testing.T) {
	const text = "let a ="
	result := parse(t, "eof.flux", text)

	d := onlyDiagnostic(t, result)
	// An unfinished construct and a wrong token are different mistakes, and
	// pointing at "the token <EOF>" helps nobody.
	if d.Code != parser.CodeUnexpectedEOF {
		t.Fatalf("code = %q, want %q", d.Code, parser.CodeUnexpectedEOF)
	}
	if !d.Primary.IsEmpty() || d.Primary.Start != len(text) {
		t.Errorf("span = %+v, want the empty span at offset %d", d.Primary, len(text))
	}
	if len(d.Notes) == 0 {
		t.Error("the diagnostic does not explain that something was left unfinished")
	}
	if strings.Contains(d.Message, "<EOF>") {
		t.Errorf("message %q exposes the parser's token name", d.Message)
	}
}

// TestFailedParseWithholdsTheTreeButKeepsItForTools prevents partial trees
// from appearing executable while preserving them for tooling.
func TestFailedParseWithholdsTheTreeButKeepsItForTools(t *testing.T) {
	result := parse(t, "partial.flux", "let a = 1\nlet b = ")

	if !result.Failed() || result.Program != nil {
		t.Fatal("a failed parse produced a program that could be executed")
	}
	tree, complete := result.Tree()
	if complete {
		t.Error("Tree reported a partial tree as complete")
	}
	if tree == nil {
		t.Fatal("no partial tree was kept for tools")
	}
	if len(tree.Statements) == 0 {
		t.Error("the partial tree kept none of the statements that did parse")
	}
}

// TestSuccessfulParseHasNoPartialTree verifies that a complete parse exposes
// only the executable tree and no errors.
func TestSuccessfulParseHasNoPartialTree(t *testing.T) {
	result := mustParse(t, "ok.flux", "let a = 1")

	if result.Partial != nil {
		t.Error("a successful parse also produced a partial tree; a caller could execute the wrong one")
	}
	tree, complete := result.Tree()
	if !complete || tree != result.Program {
		t.Error("Tree did not return the complete program")
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("a successful parse reported %v", result.Diagnostics)
	}
}

// TestTokenStreamKeepsCommentsAndWhitespace verifies that tokens and trailing
// trivia reconstruct the original fixture.
func TestTokenStreamKeepsCommentsAndWhitespace(t *testing.T) {
	const text = "// a leading note\nlet a = 1 /* inline */ \nprint(a)\n"
	result := mustParse(t, "tokens.flux", text)

	var comments, whitespace int
	var rebuilt strings.Builder
	for _, token := range result.Tokens() {
		rebuilt.WriteString(token.Value)
		switch {
		case strings.HasPrefix(token.Value, "//"), strings.HasPrefix(token.Value, "/*"):
			comments++
		case strings.TrimSpace(token.Value) == "":
			whitespace++
		}
	}
	if comments != 2 {
		t.Errorf("token stream kept %d comments, want 2", comments)
	}
	if whitespace == 0 {
		t.Error("token stream dropped whitespace, which a formatter needs")
	}
	// The stream plus the trailing trivia must reconstruct the file exactly.
	// A String token is unquoted during capture, so this program avoids one.
	if got := rebuilt.String() + result.TrailingTrivia(); got != text {
		t.Errorf("token stream reconstructs %q, want %q", got, text)
	}
	if got, want := result.TrailingTrivia(), "\n"; got != want {
		t.Errorf("trailing trivia = %q, want %q", got, want)
	}
}

// TestOnlyTheRootCapturesTokens guards the cost of the token stream. Participle
// fills a Tokens field on any node that declares one, from that node's whole
// raw range, so a Tokens field on a nested node would copy its subtree's tokens
// again for every level of nesting.
func TestOnlyTheRootCapturesTokens(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		if _, ok := typ.FieldByName("Tokens"); ok && typ != reflect.TypeOf(ast.Program{}) {
			t.Errorf("%s declares a Tokens field; only the root may capture the token stream", typ)
		}
		for i := 0; i < typ.NumField(); i++ {
			walk(typ.Field(i).Type)
		}
	}
	walk(reflect.TypeOf(ast.Program{}))
	if len(seen) < 20 {
		t.Errorf("only walked %d node types; the AST has more than that", len(seen))
	}
}

// TestParseWrapperStillWorksAndLocatesFailures checks the text-only
// compatibility API and its synthetic source location.
func TestParseWrapperStillWorksAndLocatesFailures(t *testing.T) {
	program, err := parser.Parse("let a = 1 print(a)")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(program.Statements) != 2 {
		t.Errorf("parsed %d statements, want 2", len(program.Statements))
	}

	_, err = parser.Parse("let a = 1\nlet b = let")
	if err == nil {
		t.Fatal("Parse accepted an invalid program")
	}
	if got, want := err.Error(), "<stdin>:2:9:"; !strings.HasPrefix(got, want) {
		t.Errorf("error = %q, want it to start with %q", got, want)
	}
}

// TestEmptyAndWhitespaceOnlySourcesParse checks that trivia-only files yield
// empty programs and preserve their text.
func TestEmptyAndWhitespaceOnlySourcesParse(t *testing.T) {
	for _, text := range []string{"", "\n\n", "  \t\n", "// just a comment\n"} {
		result := parse(t, "empty.flux", text)
		if result.Failed() {
			t.Errorf("parsing %q failed: %v", text, result.Diagnostics)
			continue
		}
		if len(result.Program.Statements) != 0 {
			t.Errorf("parsing %q produced %d statements, want 0", text, len(result.Program.Statements))
		}
		if got := result.TrailingTrivia(); got != text {
			t.Errorf("trailing trivia of %q = %q, want the whole text", text, got)
		}
	}
}

// TestDiagnosticCodesAreRegistered checks that parser codes have descriptions
// and belong to the syntax group.
func TestDiagnosticCodesAreRegistered(t *testing.T) {
	for _, code := range []diagnostic.Code{
		parser.CodeInvalidCharacter, parser.CodeUnexpectedToken, parser.CodeUnexpectedEOF,
	} {
		if _, ok := diagnostic.Describe(code); !ok {
			t.Errorf("%s is not registered, so it cannot appear in the code reference", code)
		}
		if group, ok := code.Group(); !ok || group != diagnostic.GroupSyntax {
			t.Errorf("%s is in group %q, want the syntax group", code, group)
		}
	}
}

// primary digs out the PrimaryExpr an expression reduces to. Expr tries Binary
// before Primary, so even a bare term arrives wrapped in a Binary with no
// comparisons and an Additive with no additions.
func primary(t *testing.T, expr *ast.Expr) *ast.PrimaryExpr {
	t.Helper()
	if expr == nil || expr.Bin == nil || expr.Bin.Left == nil {
		t.Fatalf("expression %+v is not a binary expression", expr)
	}
	return expr.Bin.Left.Left
}

// onlyDiagnostic requires exactly one parse failure and returns it for
// location and message assertions.
func onlyDiagnostic(t *testing.T, result parser.Result) diagnostic.Diagnostic {
	t.Helper()
	if !result.Failed() {
		t.Fatal("parsing succeeded, want a failure")
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %v", len(result.Diagnostics), result.Diagnostics)
	}
	return result.Diagnostics[0]
}
