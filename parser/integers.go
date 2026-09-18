package parser

import (
	"math"
	"strconv"
	"strings"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/source"
)

var CodeIntegerRange = diagnostic.Register("S_INT_RANGE", "an integer literal is outside the signed 64-bit range")

func validateIntegers(prog *ast.Program, src *source.Source) []diagnostic.Diagnostic {
	var diagnostics []diagnostic.Diagnostic
	ast.Walk(prog, func(n ast.Positioned) bool {
		if u, ok := n.(*ast.Unary); ok && u.Operator == "-" && u.Operand.Primary != nil {
			p := u.Operand.Primary
			if len(p.Postfix) == 0 && p.Base.Term != nil && p.Base.Term.Number != nil && strings.TrimLeft(p.Base.Term.Number.Text, "0") == "9223372036854775808" {
				u.MinLiteral = true
				p.Base.Term.Number.Value = math.MinInt64
				return false
			}
		}
		if t, ok := n.(*ast.Term); ok && t.Number != nil {
			v, err := strconv.ParseInt(t.Number.Text, 10, 64)
			if err != nil {
				diagnostics = append(diagnostics, diagnostic.Error(CodeIntegerRange, t.Span(src.ID()), "integer literal is outside -9223372036854775808..9223372036854775807"))
			} else {
				t.Number.Value = v
			}
		}
		return true
	})
	return diagnostics
}
