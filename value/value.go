// Package value defines Flux value operations shared by the two engines.
// It contains no AST traversal or bytecode dispatch and is bundled standalone.
package value

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pranavms13/flux-lang/fault"
)

// Cell gives a captured immutable binding an identity before its closure is
// constructed. Each execution of a declaration allocates a new cell.
type Cell struct {
	Value       any
	Initialized bool
}

const DefaultMaxDepth = 256

func Bool(v any) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fault.ConditionType(v)
}

func Unary(op string, v any) (any, error) {
	switch op {
	case "-":
		if n, ok := v.(int64); ok {
			if n == math.MinInt64 {
				return nil, fault.IntOverflow(op)
			}
			return -n, nil
		}
	case "!":
		if b, ok := v.(bool); ok {
			return !b, nil
		}
	}
	return nil, fault.UnaryType(op, v)
}

func Binary(op string, a, b any) (any, error) {
	if op == "==" || op == "!=" {
		eq, err := Equal(a, b)
		if err != nil {
			return nil, err
		}
		if op == "!=" {
			eq = !eq
		}
		return eq, nil
	}
	if op == "+" {
		if x, ok := a.(string); ok {
			if y, ok := b.(string); ok {
				return x + y, nil
			}
		}
	}
	x, okX := a.(int64)
	y, okY := b.(int64)
	if !okX || !okY {
		return nil, fault.OperandType(op, a, b)
	}
	switch op {
	case "+":
		if y > 0 && x > math.MaxInt64-y || y < 0 && x < math.MinInt64-y {
			return nil, fault.IntOverflow(op)
		}
		return x + y, nil
	case "-":
		if y < 0 && x > math.MaxInt64+y || y > 0 && x < math.MinInt64+y {
			return nil, fault.IntOverflow(op)
		}
		return x - y, nil
	case "*":
		if x == 0 || y == 0 {
			return int64(0), nil
		}
		if x == math.MinInt64 && y == -1 || y == math.MinInt64 && x == -1 {
			return nil, fault.IntOverflow(op)
		}
		z := x * y
		if z/y != x {
			return nil, fault.IntOverflow(op)
		}
		return z, nil
	case "/", "%":
		if y == 0 {
			return nil, fault.ZeroDivisor(op)
		}
		if op == "%" {
			return x % y, nil
		}
		if x == math.MinInt64 && y == -1 {
			return nil, fault.IntOverflow(op)
		}
		return x / y, nil
	case "<":
		return x < y, nil
	case "<=":
		return x <= y, nil
	case ">":
		return x > y, nil
	case ">=":
		return x >= y, nil
	}
	return nil, fault.OperandType(op, a, b)
}

func ValidKey(v any) bool {
	switch v.(type) {
	case int64, string, bool:
		return true
	}
	return false
}
func Key(v any) error {
	if !ValidKey(v) {
		return fault.InvalidKey(v)
	}
	return nil
}
func Index(v, index any) (any, error) {
	switch v := v.(type) {
	case []any:
		i, ok := index.(int64)
		if !ok {
			return nil, fault.IndexType(index)
		}
		if i < 0 || i >= int64(len(v)) {
			return nil, fault.IndexRange(i, len(v))
		}
		return v[int(i)], nil
	case map[any]any:
		if err := Key(index); err != nil {
			return nil, err
		}
		result, ok := v[index]
		if !ok {
			return nil, fault.MissingKey(index)
		}
		return result, nil
	default:
		return nil, fault.NotIndexable(v)
	}
}

// Comparable validates the whole value before equality, even when unequal
// lengths or an earlier mismatch would otherwise short circuit the comparison.
func Comparable(v any) bool {
	switch v := v.(type) {
	case nil, int64, string, bool:
		return true
	case []any:
		for _, e := range v {
			if !Comparable(e) {
				return false
			}
		}
		return true
	case map[any]any:
		for k, e := range v {
			if !ValidKey(k) || !Comparable(e) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func Equal(a, b any) (bool, error) {
	if !Comparable(a) || !Comparable(b) {
		return false, fault.Incomparable()
	}
	return equal(a, b), nil
}
func equal(a, b any) bool {
	switch a := a.(type) {
	case nil:
		return b == nil
	case int64:
		x, ok := b.(int64)
		return ok && a == x
	case string:
		x, ok := b.(string)
		return ok && a == x
	case bool:
		x, ok := b.(bool)
		return ok && a == x
	case []any:
		x, ok := b.([]any)
		if !ok || len(a) != len(x) {
			return false
		}
		for i, e := range a {
			if !equal(e, x[i]) {
				return false
			}
		}
		return true
	case map[any]any:
		x, ok := b.(map[any]any)
		if !ok || len(a) != len(x) {
			return false
		}
		for k, e := range a {
			v, ok := x[k]
			if !ok || !equal(e, v) {
				return false
			}
		}
		return true
	}
	return false
}

func Display(v any) string { return display(v, false) }
func display(v any, nested bool) string {
	switch v := v.(type) {
	case nil:
		return "<void>"
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		if nested {
			return strconv.Quote(v)
		}
		return v
	case bool:
		return strconv.FormatBool(v)
	case []any:
		parts := make([]string, len(v))
		for i, e := range v {
			parts[i] = display(e, true)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[any]any:
		parts := make([]string, 0, len(v))
		for k, e := range v {
			parts = append(parts, display(k, true)+": "+display(e, true))
		}
		sort.Strings(parts)
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		if f, ok := v.(interface{ FluxTypeName() string }); ok && f.FluxTypeName() == "function" {
			return "<function>"
		}
		return "<invalid value>"
	}
}
