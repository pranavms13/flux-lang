// parser/parser.go
package parser

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Program struct {
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Let  *LetStatement `parser:"  @@"`
	Expr *Expr         `parser:"| @@"`
}

type LetStatement struct {
	Let  string `parser:"'let'"`
	Name string `parser:"@Ident"`
	Eq   string `parser:"'='"`
	Expr *Expr  `parser:"@@"`
}

type Expr struct {
	If    *IfExpr    `parser:"  @@"`
	Func  *FuncExpr  `parser:"| @@"`
	Call  *CallExpr  `parser:"| @@"`
	Bin   *Binary    `parser:"| @@"`
	Block *BlockExpr `parser:"| @@"`
	Term  *Term      `parser:"| @@"`
}

type IfExpr struct {
	If       string `parser:"'if'"`
	Cond     *Expr  `parser:"@@"`
	Then     string `parser:"'then'"`
	ThenExpr *Expr  `parser:"@@"`
	Else     string `parser:"'else'"`
	ElseExpr *Expr  `parser:"@@"`
}

type FuncExpr struct {
	Fn     string   `parser:"'fn'"`
	Params []string `parser:"'(' (@Ident (',' @Ident)*)? ')'"`
	Arrow  string   `parser:"@Arrow"`
	Body   *Expr    `parser:"@@"`
}

type CallExpr struct {
	Name string  `parser:"@Ident"`
	Args []*Expr `parser:"'(' (@@ (',' @@)*)? ')'"`
}

type Binary struct {
	Left     *Term   `parser:"@@"`
	Operator *string `parser:"( @('+' | '-' | '==' | '<' | '>')"`
	Right    *Expr   `parser:"  @@)?"`
}

type BlockExpr struct {
	LBrace string  `parser:"'{'"`
	Exprs  []*Expr `parser:"@@*"`
	RBrace string  `parser:"'}'"`
}

type Term struct {
	Number *int    `parser:"  @Int"`
	String *string `parser:"| @String"`
	Ident  *string `parser:"| @Ident"`
	Bool   *bool   `parser:"| @Bool"`
}

var lexerRules = lexer.MustSimple([]lexer.SimpleRule{
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

var parserInstance = participle.MustBuild[Program](
	participle.Lexer(lexerRules),
	participle.Unquote("String"),
	participle.Elide("Whitespace"),
)

func Parse(input string) (*Program, error) {
	prog, err := parserInstance.ParseString("<stdin>", input)
	if err != nil {
		return nil, fmt.Errorf("Parse error: %w", err)
	}
	return prog, nil
}
