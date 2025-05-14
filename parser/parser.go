package parser

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/lexer"
)

var parserInstance = participle.MustBuild[ast.Program](
	participle.Lexer(lexer.LexerRules),
	participle.Unquote("String"),
	participle.Elide("Whitespace", "SingleLineComment", "MultiLineComment"),
)

func Parse(input string) (*ast.Program, error) {
	prog, err := parserInstance.ParseString("<stdin>", input)
	if err != nil {
		return nil, fmt.Errorf("Parse error: %w", err)
	}
	return prog, nil
}
