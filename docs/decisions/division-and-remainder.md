# D-10 — Division and remainder

- Status: accepted
- Rules: `VAL-DIVIDE`
- Phase: implemented
- Slice: P3.1 (add precedence levels and checked value operations)
- Migration: none; the operators are new

## Context

`/` and `%` are lexed as operators and appear in no grammar rule, so `6 / 2` is
a syntax error. Arithmetic without division is not arithmetic.

## Decision

Add `/` and `%` at the same precedence level as each other, binding tighter than
`+` and `-`:

- The quotient truncates toward zero.
- The remainder takes the sign of the dividend, so `a == (a / b) * b + a % b`.
- A zero divisor is an error, not an infinity or a wrapped value.
- `MinInt64 / -1` is an overflow error, because its true result is not an `int`.
  `MinInt64 % -1` is `0`.

```flux
print(7 / 2)    // 3
print(-7 / 2)   // -3, truncated toward zero
print(-7 % 2)   // -1, the sign of the dividend
```

## Consequences

The precedence table gains a level, which is the reason this decision lands with
[D-09](integers.md) rather than after it: both change the same operator code.
