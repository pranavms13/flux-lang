package ast

type ListExpr struct {
	LBrack string  `parser:"'['"`
	Elems  []*Expr `parser:"(@@ (',' @@)*)?"`
	RBrack string  `parser:"']'"`
}

type DictExpr struct {
	LBrace string      `parser:"'{'"`
	Pairs  []*DictPair `parser:"(@@ (',' @@)*)?"`
	RBrace string      `parser:"'}'"`
}

type DictPair struct {
	Key   *Expr  `parser:"@@"`
	Colon string `parser:"':'"`
	Value *Expr  `parser:"@@"`
}

type BlockExpr struct {
	LBrace string  `parser:"'{'"`
	Exprs  []*Expr `parser:"@@*"`
	RBrace string  `parser:"'}'"`
}

type IfExpr struct {
	If       string `parser:"'if'"`
	Cond     *Expr  `parser:"@@"`
	Then     string `parser:"'then'"`
	ThenExpr *Expr  `parser:"@@"`
	Else     string `parser:"'else'"`
	ElseExpr *Expr  `parser:"@@"`
}

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
	Term  *Term      `parser:"  @@"`
	Group *GroupExpr `parser:"| @@"`
	List  *ListExpr  `parser:"| @@"`
	Dict  *DictExpr  `parser:"| @@"`
	Block *BlockExpr `parser:"| @@"`
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
	Let      string    `parser:"'let'"`
	Name     string    `parser:"@Ident"`
	TypeAnno *TypeAnno `parser:"@@?"`
	Eq       string    `parser:"'='"`
	Expr     *Expr     `parser:"@@"`
}

type TypeAnno struct {
	Colon string `parser:"':'"`
	Type  *Type  `parser:"@@"`
}

type Type struct {
	Basic    *string   `parser:"  @('int' | 'string' | 'bool' | 'void')"`
	List     *ListType `parser:"| @@"`
	Dict     *DictType `parser:"| @@"`
	Function *FuncType `parser:"| @@"`
}

type ListType struct {
	LBrack   string `parser:"'['"`
	ElemType *Type  `parser:"@@"`
	RBrack   string `parser:"']'"`
}

type DictType struct {
	LBrace    string `parser:"'{'"`
	KeyType   *Type  `parser:"@@"`
	Colon     string `parser:"':'"`
	ValueType *Type  `parser:"@@"`
	RBrace    string `parser:"'}'"`
}

type FuncType struct {
	Fn         string  `parser:"'fn'"`
	LParen     string  `parser:"'('"`
	ParamTypes []*Type `parser:"(@@ (',' @@)*)?"`
	RParen     string  `parser:"')'"`
	Arrow      string  `parser:"@TypeArrow"`
	ReturnType *Type   `parser:"@@"`
}

type Term struct {
	Number *int     `parser:"  @Int"`
	String *string  `parser:"| @String"`
	Ident  *string  `parser:"| @Ident"`
	Bool   *Boolean `parser:"| @Bool"`
}

type CallExpr struct {
	LParen string  `parser:"'('"`
	Args   []*Expr `parser:"(@@ (',' @@)*)?"`
	RParen string  `parser:"')'"`
}

// Binary separates comparisons from addition/subtraction to preserve precedence.
type Binary struct {
	Left *Additive     `parser:"@@"`
	Rest []*Comparison `parser:"@@*"`
}

type Comparison struct {
	Operator string    `parser:"@('==' | '<' | '>')"`
	Right    *Additive `parser:"@@"`
}

type Additive struct {
	Left *PrimaryExpr `parser:"@@"`
	Rest []*Addition  `parser:"@@*"`
}

type Addition struct {
	Operator string       `parser:"@('+' | '-')"`
	Right    *PrimaryExpr `parser:"@@"`
}

type GroupExpr struct {
	Expr *Expr `parser:"'(' @@ ')'"`
}

// Boolean captures the literal's value, rather than whether a token was present.
type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true" || values[0] == "yes"
	return nil
}

type IndexExpr struct {
	LBrack string `parser:"'['"`
	Index  *Expr  `parser:"@@"`
	RBrack string `parser:"']'"`
}
