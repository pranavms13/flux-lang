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
	Term *Term     `parser:"  @@"`
	List *ListExpr `parser:"| @@"`
	Dict *DictExpr `parser:"| @@"`
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
