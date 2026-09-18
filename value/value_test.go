package value_test

import (
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/value"
	"math"
	"math/big"
	"testing"
)

// The oracle uses arbitrary precision rather than mirroring the overflow guards.
func arithmetic(t *testing.T, a, b int64, op string) {
	t.Helper()
	x, y := big.NewInt(a), big.NewInt(b)
	expected := new(big.Int)
	got, err := value.Binary(op, a, b)
	if (op == "/" || op == "%") && b == 0 {
		if e, ok := err.(*fault.Error); !ok || e.Code != fault.CodeZeroDivisor {
			t.Fatalf("%d%s%d: %v", a, op, b, err)
		}
		return
	}
	switch op {
	case "+":
		expected.Add(x, y)
	case "-":
		expected.Sub(x, y)
	case "*":
		expected.Mul(x, y)
	case "/":
		expected.Quo(x, y)
	case "%":
		expected.Rem(x, y)
	}
	if !expected.IsInt64() {
		if e, ok := err.(*fault.Error); !ok || e.Code != fault.CodeIntOverflow {
			t.Fatalf("%d%s%d: got %v,%v; want overflow", a, op, b, got, err)
		}
		return
	}
	if err != nil || got != expected.Int64() {
		t.Fatalf("%d%s%d: %v,%v; want %s", a, op, b, got, err, expected)
	}
}
func TestCheckedArithmetic(t *testing.T) {
	nums := []int64{math.MinInt64, math.MinInt64 + 1, -3037000500, -2, -1, 0, 1, 2, 3037000500, math.MaxInt64 - 1, math.MaxInt64}
	for _, a := range nums {
		for _, b := range nums {
			for _, op := range []string{"+", "-", "*", "/", "%"} {
				arithmetic(t, a, b, op)
			}
		}
	}
}
func FuzzArithmetic(f *testing.F) {
	f.Add(int64(math.MinInt64), int64(-1))
	f.Add(int64(math.MaxInt64), int64(2))
	f.Add(int64(-7), int64(3))
	f.Add(int64(0), int64(0))
	f.Fuzz(func(t *testing.T, a, b int64) {
		for _, op := range []string{"+", "-", "*", "/", "%"} {
			arithmetic(t, a, b, op)
		}
	})
}
