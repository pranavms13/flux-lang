package lexer_test

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	fluxlexer "github.com/pranavms13/flux-lang/lexer"
)

func TestLexer(t *testing.T) {
	symbols := fluxlexer.LexerRules.Symbols()
	tests := []struct {
		name     string
		input    string
		expected []lexer.Token
	}{
		{
			name:  "Basic tokens",
			input: "let x = 42",
			expected: []lexer.Token{
				{Type: symbols["Keywords"], Value: "let"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "x"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "="},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Int"], Value: "42"},
			},
		},
		{
			name:  "String literal",
			input: `let name = "Flux"`,
			expected: []lexer.Token{
				{Type: symbols["Keywords"], Value: "let"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "name"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "="},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["String"], Value: `"Flux"`},
			},
		},
		{
			name:  "Function definition",
			input: "let add = fn(x, y) => x + y",
			expected: []lexer.Token{
				{Type: symbols["Keywords"], Value: "let"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "add"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "="},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Keywords"], Value: "fn"},
				{Type: symbols["Operators"], Value: "("},
				{Type: symbols["Ident"], Value: "x"},
				{Type: symbols["Operators"], Value: ","},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "y"},
				{Type: symbols["Operators"], Value: ")"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Arrow"], Value: "=>"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "x"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "+"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "y"},
			},
		},
		{
			name:  "If expression",
			input: "if x > 0 then { print(x) } else { print(0) }",
			expected: []lexer.Token{
				{Type: symbols["Keywords"], Value: "if"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "x"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: ">"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Int"], Value: "0"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Keywords"], Value: "then"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "{"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "print"},
				{Type: symbols["Operators"], Value: "("},
				{Type: symbols["Ident"], Value: "x"},
				{Type: symbols["Operators"], Value: ")"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "}"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Keywords"], Value: "else"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "{"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Ident"], Value: "print"},
				{Type: symbols["Operators"], Value: "("},
				{Type: symbols["Int"], Value: "0"},
				{Type: symbols["Operators"], Value: ")"},
				{Type: symbols["Whitespace"], Value: " "},
				{Type: symbols["Operators"], Value: "}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex, _ := fluxlexer.LexerRules.Lex("", strings.NewReader(tt.input))
			var tokens []lexer.Token
			for {
				tok, err := lex.Next()
				if err != nil {
					t.Fatalf("Lexer error: %v", err)
				}
				if tok.EOF() {
					break
				}
				tokens = append(tokens, tok)
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("Token %d: expected type %v, got %v", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("Token %d: expected value %s, got %s", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestPhase3LongestMatch(t *testing.T) {
	lex, err := fluxlexer.LexerRules.Lex("operators.flux", strings.NewReader("1-2<=3>=4!=5==6&&true||false;!true=>->"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1", "-", "2", "<=", "3", ">=", "4", "!=", "5", "==", "6", "&&", "true", "||", "false", ";", "!", "true", "=>", "->"}
	for _, value := range want {
		tok, err := lex.Next()
		if err != nil || tok.Value != value {
			t.Fatalf("token=%v err=%v, want %q", tok, err, value)
		}
	}
	tok, err := lex.Next()
	if err != nil || !tok.EOF() {
		t.Fatalf("expected EOF, got %v %v", tok, err)
	}
}
