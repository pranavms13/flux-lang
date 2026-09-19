package ast

// Enhanced function parameter with optional type annotation
type FuncParam struct {
	Node
	Name     string    `parser:"@Ident"`
	TypeAnno *TypeAnno `parser:"@@?"`
}

// Enhanced function expression with type annotations
type FuncExpr struct {
	Node
	Fn         string       `parser:"'fn':Keywords"`
	LParen     string       `parser:"'(':Operators"`
	Params     []*FuncParam `parser:"(@@ (',':Operators @@)*)?"`
	RParen     string       `parser:"')':Operators"`
	ReturnAnno *TypeAnno    `parser:"@@?"`
	Arrow      string       `parser:"@Arrow"`
	Body       *Expr        `parser:"@@"`
}
