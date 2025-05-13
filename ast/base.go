package ast

type Expr struct {
	If    *IfExpr    `parser:"  @@"`
	Func  *FuncExpr  `parser:"| @@"`
	Call  *CallExpr  `parser:"| @@"`
	Bin   *Binary    `parser:"| @@"`
	Block *BlockExpr `parser:"| @@"`
	Term  *Term      `parser:"| @@"`
	List  *ListExpr  `parser:"| @@"`
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
	Name string  `parser:"@Ident"`
	Args []*Expr `parser:"'(' (@@ (',' @@)*)? ')'"`
}

type Binary struct {
	Left     *Term   `parser:"@@"`
	Operator *string `parser:"( @('+' | '-' | '==' | '<' | '>')"`
	Right    *Expr   `parser:"  @@)?"`
}
