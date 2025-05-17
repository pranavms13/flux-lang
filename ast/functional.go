package ast

type FuncExpr struct {
	Fn     string   `parser:"'fn'"`
	Params []string `parser:"'(' (@Ident (',' @Ident)*)? ')'"`
	Arrow  string   `parser:"@Arrow"`
	Body   *Expr    `parser:"@@"`
}
