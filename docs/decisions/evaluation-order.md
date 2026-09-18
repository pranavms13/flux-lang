# D-05 — Evaluation order

- Status: accepted
- Rules: `EVL-ORDER-STATEMENT`, `EVL-ORDER-CALL`, `EVL-ORDER-LIST`, `EVL-ORDER-DICT`, `EVL-IF-BRANCH`
- Phase: implemented
- Slice: none; this records existing behavior
- Migration: none

## Context

Both engines evaluate left to right and callee before arguments, and neither
documented it. An undocumented order is one an optimisation is free to change,
and the programs that notice are the ones with side effects — which, in a
language whose only effect is `print`, means the programs people write to
understand what happened.

## Decision

Fix the order both engines already use, and test it:

- Statements run in source order.
- A call evaluates the callee, then the arguments left to right, then calls.
- List elements evaluate left to right.
- Dictionary pairs evaluate in source order, key before value.
- A conditional evaluates its condition and exactly one branch.

```flux
let trace = fn(v) => { print(v) v }
let d = {trace("k1"): trace("v1"), trace("k2"): trace("v2")}
// prints k1, v1, k2, v2
```

## Consequences

`compileBlock` and `compilePrimary` must keep emitting in this order, and
`evalPrimary` must keep applying postfixes in source order. The conformance
fixtures fail loudly if either is reordered for speed.
