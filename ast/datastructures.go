package ast

import "github.com/alecthomas/participle/v2/lexer"

type ListExpr struct {
	Node
	LBrack string  `parser:"'[':Operators"`
	Elems  []*Expr `parser:"(@@ (',':Operators @@)*)?"`
	RBrack string  `parser:"']':Operators"`
}

type DictExpr struct {
	Node
	LBrace string      `parser:"'{':Operators"`
	Pairs  []*DictPair `parser:"(@@ (',':Operators @@)*)?"`
	RBrace string      `parser:"'}':Operators"`
}

type DictPair struct {
	Node
	Key   *Expr  `parser:"@@"`
	Colon string `parser:"':':Operators"`
	Value *Expr  `parser:"@@"`
}

type BlockExpr struct {
	Node
	LBrace     string       `parser:"'{':Operators"`
	Statements []*Statement `parser:"@@*"`
	RBrace     string       `parser:"'}':Operators"`
}

type IfExpr struct {
	Node
	If       string `parser:"'if':Keywords"`
	Cond     *Expr  `parser:"@@"`
	Then     string `parser:"'then':Keywords"`
	ThenExpr *Expr  `parser:"@@"`
	Else     string `parser:"'else':Keywords"`
	ElseExpr *Expr  `parser:"@@"`
}

type Expr struct {
	Node
	If   *IfExpr   `parser:"  @@"`
	Func *FuncExpr `parser:"| @@"`
	Bin  *Binary   `parser:"| @@"`
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
	Let       *LetStatement `parser:"( @@"`
	Expr      *Expr         `parser:"| @@ )"`
	Separator *Separator    `parser:"@@?"`
}

type LetStatement struct {
	Node
	Let      string    `parser:"'let':Keywords"`
	Name     string    `parser:"@Ident"`
	TypeAnno *TypeAnno `parser:"@@?"`
	Eq       string    `parser:"'=':Operators"`
	Expr     *Expr     `parser:"@@"`
}

type TypeAnno struct {
	Node
	Colon string `parser:"':':Operators"`
	Type  *Type  `parser:"@@"`
}

type Type struct {
	Node
	Basic    *string   `parser:"  @('int':Keywords | 'string':Keywords | 'bool':Keywords | 'void':Keywords)"`
	List     *ListType `parser:"| @@"`
	Dict     *DictType `parser:"| @@"`
	Function *FuncType `parser:"| @@"`
}

type ListType struct {
	Node
	LBrack   string `parser:"'[':Operators"`
	ElemType *Type  `parser:"@@"`
	RBrack   string `parser:"']':Operators"`
}

type DictType struct {
	Node
	LBrace    string `parser:"'{':Operators"`
	KeyType   *Type  `parser:"@@"`
	Colon     string `parser:"':':Operators"`
	ValueType *Type  `parser:"@@"`
	RBrace    string `parser:"'}':Operators"`
}

type FuncType struct {
	Node
	Fn         string  `parser:"'fn':Keywords"`
	LParen     string  `parser:"'(':Operators"`
	ParamTypes []*Type `parser:"(@@ (',':Operators @@)*)?"`
	RParen     string  `parser:"')':Operators"`
	Arrow      string  `parser:"@TypeArrow"`
	ReturnType *Type   `parser:"@@"`
}

type Term struct {
	Node
	Number *Integer `parser:"  @Int"`
	String *string  `parser:"| @String"`
	Ident  *string  `parser:"| @Ident"`
	Bool   *Boolean `parser:"| @Bool"`
}

type CallExpr struct {
	Node
	LParen string  `parser:"'(':Operators"`
	Args   []*Expr `parser:"(@@ (',':Operators @@)*)?"`
	RParen string  `parser:"')':Operators"`
}

// Each level is non-left-recursive. Relational permits only one comparison.
type Binary struct {
	Node
	Left *LogicalAnd    `parser:"@@"`
	Rest []*Disjunction `parser:"@@*"`
}
type Disjunction struct {
	Node
	Operator string      `parser:"@'||':Operators"`
	Right    *LogicalAnd `parser:"@@"`
}
type LogicalAnd struct {
	Node
	Left *Equality      `parser:"@@"`
	Rest []*Conjunction `parser:"@@*"`
}
type Conjunction struct {
	Node
	Operator string    `parser:"@'&&':Operators"`
	Right    *Equality `parser:"@@"`
}
type Equality struct {
	Node
	Left *Relational   `parser:"@@"`
	Rest []*EqualityOp `parser:"@@*"`
}
type EqualityOp struct {
	Node
	Operator string      `parser:"@('==':Operators | '!=':Operators)"`
	Right    *Relational `parser:"@@"`
}
type Relational struct {
	Node
	Left *Additive   `parser:"@@"`
	Rest *Comparison `parser:"@@?"`
}
type Comparison struct {
	Node
	Operator string    `parser:"@('<=':Operators | '>=':Operators | '<':Operators | '>':Operators)"`
	Right    *Additive `parser:"@@"`
}
type Additive struct {
	Node
	Left *Multiplicative `parser:"@@"`
	Rest []*Addition     `parser:"@@*"`
}
type Addition struct {
	Node
	Operator string          `parser:"@('+':Operators | '-':Operators)"`
	Right    *Multiplicative `parser:"@@"`
}
type Multiplicative struct {
	Node
	Left *Unary            `parser:"@@"`
	Rest []*Multiplication `parser:"@@*"`
}
type Multiplication struct {
	Node
	Operator string `parser:"@('*':Operators | '/':Operators | '%':Operators)"`
	Right    *Unary `parser:"@@"`
}
type Unary struct {
	Node
	Operator string       `parser:"( @('-':Operators | '!':Operators)"`
	Operand  *Unary       `parser:"@@ )"`
	Primary  *PrimaryExpr `parser:"| @@"`
	// MinLiteral represents the signed minimum literal, whose positive
	// magnitude is not itself a Flux int. Filled by parser validation.
	MinLiteral bool
}

// Integer preserves the magnitude token while parsing. Value is int64 after
// validation; Text lets the parser diagnose range errors without host-width
// conversions and recognize the signed minimum beneath unary minus.
type Integer struct {
	Text  string
	Value int64
}

func (i *Integer) Capture(values []string) error { i.Text = values[0]; return nil }

type Separator struct {
	Node
	Semicolon string `parser:"';':Operators"`
}

type GroupExpr struct {
	Node
	Expr *Expr `parser:"'(':Operators @@ ')':Operators"`
}

// Boolean captures the literal's value, rather than whether a token was present.
type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true" || values[0] == "yes"
	return nil
}

type IndexExpr struct {
	Node
	LBrack string `parser:"'[':Operators"`
	Index  *Expr  `parser:"@@"`
	RBrack string `parser:"']':Operators"`
}
