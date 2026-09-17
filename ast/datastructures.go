package ast

import "github.com/alecthomas/participle/v2/lexer"

type ListExpr struct {
	Node
	LBrack string  `parser:"'['"`
	Elems  []*Expr `parser:"(@@ (',' @@)*)?"`
	RBrack string  `parser:"']'"`
}

type DictExpr struct {
	Node
	LBrace string      `parser:"'{'"`
	Pairs  []*DictPair `parser:"(@@ (',' @@)*)?"`
	RBrace string      `parser:"'}'"`
}

type DictPair struct {
	Node
	Key   *Expr  `parser:"@@"`
	Colon string `parser:"':'"`
	Value *Expr  `parser:"@@"`
}

type BlockExpr struct {
	Node
	LBrace string  `parser:"'{'"`
	Exprs  []*Expr `parser:"@@*"`
	RBrace string  `parser:"'}'"`
}

type IfExpr struct {
	Node
	If       string `parser:"'if'"`
	Cond     *Expr  `parser:"@@"`
	Then     string `parser:"'then'"`
	ThenExpr *Expr  `parser:"@@"`
	Else     string `parser:"'else'"`
	ElseExpr *Expr  `parser:"@@"`
}

type Expr struct {
	Node
	If      *IfExpr      `parser:"  @@"`
	Func    *FuncExpr    `parser:"| @@"`
	Bin     *Binary      `parser:"| @@"`
	Block   *BlockExpr   `parser:"| @@"`
	Primary *PrimaryExpr `parser:"| @@"`
}

type PrimaryExpr struct {
	Node
	Base    *BaseExpr  `parser:"@@"`
	Postfix []*Postfix `parser:"@@*"`
}

type BaseExpr struct {
	Node
	Term  *Term      `parser:"  @@"`
	Group *GroupExpr `parser:"| @@"`
	List  *ListExpr  `parser:"| @@"`
	Dict  *DictExpr  `parser:"| @@"`
	Block *BlockExpr `parser:"| @@"`
}

type Postfix struct {
	Node
	Call  *CallExpr  `parser:"  @@"`
	Index *IndexExpr `parser:"| @@"`
}

type Program struct {
	Node
	// Tokens is the whole token stream the parser consumed, including the
	// comments and whitespace elided from the tree. Participle fills it on any
	// struct that declares it, from the raw token range of that node, so
	// declaring it on the root captures the file once rather than copying each
	// nested range into the node that encloses it.
	//
	// The stream ends at the last token the parser consumed. Whatever follows
	// EndPos is trivia by definition — a real token there would have been
	// consumed or reported — so a formatter recovers it from the source
	// snapshot rather than from this slice.
	Tokens     []lexer.Token
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Node
	Let  *LetStatement `parser:"  @@"`
	Expr *Expr         `parser:"| @@"`
}

type LetStatement struct {
	Node
	Let      string    `parser:"'let'"`
	Name     string    `parser:"@Ident"`
	TypeAnno *TypeAnno `parser:"@@?"`
	Eq       string    `parser:"'='"`
	Expr     *Expr     `parser:"@@"`
}

type TypeAnno struct {
	Node
	Colon string `parser:"':'"`
	Type  *Type  `parser:"@@"`
}

type Type struct {
	Node
	Basic    *string   `parser:"  @('int' | 'string' | 'bool' | 'void')"`
	List     *ListType `parser:"| @@"`
	Dict     *DictType `parser:"| @@"`
	Function *FuncType `parser:"| @@"`
}

type ListType struct {
	Node
	LBrack   string `parser:"'['"`
	ElemType *Type  `parser:"@@"`
	RBrack   string `parser:"']'"`
}

type DictType struct {
	Node
	LBrace    string `parser:"'{'"`
	KeyType   *Type  `parser:"@@"`
	Colon     string `parser:"':'"`
	ValueType *Type  `parser:"@@"`
	RBrace    string `parser:"'}'"`
}

type FuncType struct {
	Node
	Fn         string  `parser:"'fn'"`
	LParen     string  `parser:"'('"`
	ParamTypes []*Type `parser:"(@@ (',' @@)*)?"`
	RParen     string  `parser:"')'"`
	Arrow      string  `parser:"@TypeArrow"`
	ReturnType *Type   `parser:"@@"`
}

type Term struct {
	Node
	Number *int     `parser:"  @Int"`
	String *string  `parser:"| @String"`
	Ident  *string  `parser:"| @Ident"`
	Bool   *Boolean `parser:"| @Bool"`
}

type CallExpr struct {
	Node
	LParen string  `parser:"'('"`
	Args   []*Expr `parser:"(@@ (',' @@)*)?"`
	RParen string  `parser:"')'"`
}

// Binary separates comparisons from addition/subtraction to preserve precedence.
type Binary struct {
	Node
	Left *Additive     `parser:"@@"`
	Rest []*Comparison `parser:"@@*"`
}

type Comparison struct {
	Node
	Operator string    `parser:"@('==' | '<' | '>')"`
	Right    *Additive `parser:"@@"`
}

type Additive struct {
	Node
	Left *PrimaryExpr `parser:"@@"`
	Rest []*Addition  `parser:"@@*"`
}

type Addition struct {
	Node
	Operator string       `parser:"@('+' | '-')"`
	Right    *PrimaryExpr `parser:"@@"`
}

type GroupExpr struct {
	Node
	Expr *Expr `parser:"'(' @@ ')'"`
}

// Boolean captures the literal's value, rather than whether a token was present.
type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true" || values[0] == "yes"
	return nil
}

type IndexExpr struct {
	Node
	LBrack string `parser:"'['"`
	Index  *Expr  `parser:"@@"`
	RBrack string `parser:"']'"`
}
