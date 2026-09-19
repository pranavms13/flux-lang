package ast

// Operation describes one application in a parsed precedence chain. At starts
// at the operator; Left is accumulated by the consumer, preserving associativity.
type Operation struct {
	At       Node
	Operator string
	Right    Positioned
}

// Chain exposes the precedence grammar to tree visitors. Execution and type
// rules remain in their respective consumers, including short circuiting.
func Chain(n Positioned) (Positioned, []Operation) {
	var left Positioned
	var rest []Operation
	switch n := n.(type) {
	case *Binary:
		left = n.Left
		for _, r := range n.Rest {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	case *LogicalAnd:
		left = n.Left
		for _, r := range n.Rest {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	case *Equality:
		left = n.Left
		for _, r := range n.Rest {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	case *Relational:
		left = n.Left
		if r := n.Rest; r != nil {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	case *Additive:
		left = n.Left
		for _, r := range n.Rest {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	case *Multiplicative:
		left = n.Left
		for _, r := range n.Rest {
			rest = append(rest, Operation{r.Node, r.Operator, r.Right})
		}
	}
	return left, rest
}

// Walk visits syntax nodes in source order. Returning false prunes a subtree.
// It does not follow analysis metadata or mutate the tree.
func Walk(n Positioned, visit func(Positioned) bool) {
	if n == nil || !visit(n) {
		return
	}
	walk := func(n Positioned) { Walk(n, visit) }
	if left, rest := Chain(n); left != nil {
		walk(left)
		for _, r := range rest {
			walk(r.Right)
		}
		return
	}
	switch n := n.(type) {
	case *Program:
		for _, s := range n.Statements {
			walk(s)
		}
	case *Statement:
		if n.Let != nil {
			walk(n.Let)
		} else if n.Expr != nil {
			walk(n.Expr)
		}
	case *LetStatement:
		walk(n.Expr)
	case *Expr:
		if n.If != nil {
			walk(n.If)
		} else if n.Func != nil {
			walk(n.Func)
		} else {
			walk(n.Bin)
		}
	case *IfExpr:
		walk(n.Cond)
		walk(n.ThenExpr)
		walk(n.ElseExpr)
	case *FuncExpr:
		for _, p := range n.Params {
			walk(p)
		}
		walk(n.Body)
	case *BlockExpr:
		for _, s := range n.Statements {
			walk(s)
		}
	case *Unary:
		if n.Operand != nil {
			walk(n.Operand)
		} else {
			walk(n.Primary)
		}
	case *PrimaryExpr:
		walk(n.Base)
		for _, p := range n.Postfix {
			walk(p)
		}
	case *BaseExpr:
		switch {
		case n.Term != nil:
			walk(n.Term)
		case n.Group != nil:
			walk(n.Group.Expr)
		case n.Block != nil:
			walk(n.Block)
		case n.List != nil:
			walk(n.List)
		case n.Dict != nil:
			walk(n.Dict)
		}
	case *Postfix:
		if n.Call != nil {
			walk(n.Call)
		} else {
			walk(n.Index)
		}
	case *CallExpr:
		for _, e := range n.Args {
			walk(e)
		}
	case *IndexExpr:
		walk(n.Index)
	case *ListExpr:
		for _, e := range n.Elems {
			walk(e)
		}
	case *DictExpr:
		for _, p := range n.Pairs {
			walk(p.Key)
			walk(p.Value)
		}
	}
}

// FunctionLiteral unwraps parentheses and precedence-only nodes. It accepts
// only a literal, never a call or a conditional that happens to return one.
func FunctionLiteral(n Positioned) *FuncExpr {
	for n != nil {
		if left, rest := Chain(n); left != nil {
			if len(rest) != 0 {
				return nil
			}
			n = left
			continue
		}
		switch x := n.(type) {
		case *FuncExpr:
			return x
		case *Expr:
			if x.Func != nil {
				return x.Func
			}
			if x.If != nil {
				return nil
			}
			n = x.Bin
		case *Unary:
			if x.Operator != "" {
				return nil
			}
			n = x.Primary
		case *PrimaryExpr:
			if len(x.Postfix) != 0 {
				return nil
			}
			n = x.Base
		case *BaseExpr:
			if x.Group == nil {
				return nil
			}
			n = x.Group.Expr
		default:
			return nil
		}
	}
	return nil
}
