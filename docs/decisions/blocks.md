# D-02 — Blocks and their result

- Status: accepted
- Rules: `GRM-BLOCK`, `GRM-BLOCK-DECLARATION`, `EVL-BLOCK-RESULT`
- Phase: implemented
- Slice: P3.3 (resolve local bindings and capture environments)
- Migration: none; the change is additive

## Context

A block is a sequence of expressions in braces whose value is the last one. It
cannot contain a declaration, so there is no way to name an intermediate result
inside one, which pushes every temporary out to the top level.

## Decision

Permit `let` inside a block. A declaration is scoped to the block that contains
it. A block whose last item is a declaration produces void, because a
declaration has no value.

```flux
// today: a temporary has to become a global
let doubled = x + x
let result = { print(doubled) doubled }

// phase 3
let result = { let doubled = x + x
               print(doubled)
               doubled }
```

## Consequences

Nothing that parses today changes meaning: a block that contains no declaration
behaves exactly as before. Block scoping is what makes `BND-CAPTURE-IDENTITY`
observable, so the two land together.
