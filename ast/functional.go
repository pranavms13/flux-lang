package ast

// Enhanced function parameter with optional type annotation
type FuncParam struct {
	Name     string    `parser:"@Ident"`
	TypeAnno *TypeAnno `parser:"@@?"`
}

// Enhanced function expression with type annotations
type FuncExpr struct {
	Fn         string       `parser:"'fn'"`
	LParen     string       `parser:"'('"`
	Params     []*FuncParam `parser:"(@@ (',' @@)*)?"`
	RParen     string       `parser:"')'"`
	ReturnAnno *TypeAnno    `parser:"@@?"`
	Arrow      string       `parser:"@Arrow"`
	Body       *Expr        `parser:"@@"`
}
