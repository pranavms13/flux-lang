# D-17 — Inference constrains unannotated parameters

- Status: accepted
- Rules: `TYP-INFERRED`, `TYP-INFERENCE-CONSTRAINTS`
- Phase: 4
- Slice: P4.1–P4.2 (separate type identity and solve constraints)
- Migration: [Unannotated parameters](../MIGRATION.md#unannotated-parameters)

## Context

An unannotated parameter is given `UnknownType`, which compares equal to every
type. That is not inference; it is a way of declining to check. It also makes
type equality non-symmetric, which `TypesEqual` currently papers over by trying
the comparison both ways.

```flux
let f = fn(x) => x + 1
print(f("a"))   // accepted, then fails at run time
```

## Decision

Give an unannotated parameter a type variable, and constrain it from its uses.
`fn(x) => x + 1` is known to take and return `int`, so the call above is rejected
where it is written.

A type variable is a distinct thing from a recovery placeholder, and both are
distinct from a value whose type is genuinely dynamic. Phase 4 separates the
three, because conflating them is what makes `UnknownType` compare equal to
everything.

## Consequences

Programs that pass a wrongly typed argument to an unannotated function start
failing the checker instead of at run time — earlier, and at the argument rather
than inside the body. This is the one decision in this set whose phase is 4; it
is recorded now because [D-08](forward-references.md) commits to typing a
recursive function from its annotation, and Phase 4 has to keep that working
without one.
