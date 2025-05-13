package lexer

import "github.com/alecthomas/participle/v2/lexer"

var LexerRules = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "MultiLineComment", Pattern: `/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`},
	{Name: "SingleLineComment", Pattern: `//[^\n]*`},
	{Name: "Bool", Pattern: `true|false`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
	{Name: "Int", Pattern: `[0-9]+`},
	{Name: "String", Pattern: `"[^"\\]*(?:\\.[^"\\]*)*"`},
	{Name: "Arrow", Pattern: `=>`},
	{Name: "Operators", Pattern: `==|[=+\-<>(){},]`},
	{Name: "Keywords", Pattern: `\b(if|then|else)\b`},
	{Name: "Whitespace", Pattern: `[ \t\n\r]+`},
})
