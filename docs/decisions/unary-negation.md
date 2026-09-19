# D-11 — Unary negation

- Status: accepted
- Rules: `VAL-NEG`
- Phase: implemented
- Slice: P3.1 (add precedence levels and checked value operations)
- Migration: none; the operator is new

## Context

There is no unary minus and no negative literal, so `-5` is a syntax error and a
negative number has to be written `0 - 5`. This is not in the plan's decision
table; it came out of writing the specification, where a grammar with
subtraction and no negation could not be stated without looking like an omission.

## Decision

Add prefix `-` on an `int`, binding tighter than any binary operator and looser
than a postfix call or index, so `-f(x)` negates the result of the call.

A negative literal stays out of the lexer: `-5` is negation applied to `5`. One
rule is easier to state than two that overlap.

```flux
print(-5)      // phase 3
print(0 - 5)   // today
```

## Consequences

`-MinInt64` overflows, and is reported the same way as any other overflow under
[D-09](integers.md).
