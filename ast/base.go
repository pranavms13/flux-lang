package ast

type Expr struct {
	If      *IfExpr      `parser:"  @@"`
	Func    *FuncExpr    `parser:"| @@"`
	Bin     *Binary      `parser:"| @@"`
	Block   *BlockExpr   `parser:"| @@"`
	Primary *PrimaryExpr `parser:"| @@"`
}

type PrimaryExpr struct {
	Base    *BaseExpr  `parser:"@@"`
	Postfix []*Postfix `parser:"@@*"`
}

type BaseExpr struct {
	Term *Term     `parser:"  @@"`
	List *ListExpr `parser:"| @@"`
}

type Postfix struct {
	Call  *CallExpr  `parser:"  @@"`
	Index *IndexExpr `parser:"| @@"`
}

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

type Term struct {
	Number *int    `parser:"  @Int"`
	String *string `parser:"| @String"`
	Ident  *string `parser:"| @Ident"`
	Bool   *bool   `parser:"| @Bool"`
}

type CallExpr struct {
	LParen string  `parser:"'('"`
	Args   []*Expr `parser:"(@@ (',' @@)*)?"`
	RParen string  `parser:"')'"`
}

type Binary struct {
	Left     *PrimaryExpr `parser:"@@"`
	Operator *string      `parser:"( @('+' | '-' | '==' | '<' | '>')"`
	Right    *Expr        `parser:"  @@)?"`
}

type IndexExpr struct {
	LBrack string `parser:"'['"`
	Index  *Expr  `parser:"@@"`
	RBrack string `parser:"']'"`
}
