# D-13 — Equality

- Status: accepted
- Rules: `VAL-EQUALITY`, `VAL-EQUALITY-FUNCTION`
- Phase: 3
- Slice: P3.1 (add precedence levels and checked value operations)
- Migration: [Comparing functions](../MIGRATION.md#comparing-functions)

## Context

`==` is implemented with the host's deep structural comparison. For scalars and
containers that is exactly right. For functions it is an accident: a function
equals itself because the comparison reaches the same pointer, and differs from
an identical literal because it reaches a different one.

```flux
let f = fn(x) => x
let g = fn(x) => x
print(f == g)   // false
print(f == f)   // true
```

Neither answer means anything. There is no definition of equality for functions
that a program could rely on, so there should be no answer at all.

## Decision

`==` compares scalars by value and containers element by element. Comparing two
functions, or two containers that hold functions, is rejected as
`T_INCOMPARABLE`. Values of different types are never equal.

The comparison is implemented over Flux values rather than by reflecting over
host values, so what is comparable is decided by this rule and not by what the
host happens to support.

## Consequences

The checker learns which types are comparable, which is a prerequisite for using
a value as a dictionary key ([D-14](collections.md)). Programs that compare
functions stop compiling; there were none that meant anything by it.
