package ast

type ListExpr struct {
	LBrack string  `parser:"'['"`
	Elems  []*Expr `parser:"(@@ (',' @@)*)?"`
	RBrack string  `parser:"']'"`
}
