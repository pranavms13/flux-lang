package lexer

import "github.com/alecthomas/participle/v2/lexer"

var LexerRules = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "SingleLineComment", Pattern: `//[^\n]*`},
	{Name: "MultiLineComment", Pattern: `/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`},
	{Name: "Arrow", Pattern: `=>`},
	{Name: "TypeArrow", Pattern: `->`},
	{Name: "Keywords", Pattern: `\b(if|then|else|let|fn|int|string|bool|void)\b`},
	{Name: "Bool", Pattern: `\b(true|false|yes|no)\b`},
	{Name: "String", Pattern: `"[^"]*"`},
	{Name: "Int", Pattern: `\d+`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
	{Name: "Operators", Pattern: `==|[+\-*/%<>=!&|(){}\[\],:]`},
	{Name: "Whitespace", Pattern: `[ \t\n\r]+`},
})
