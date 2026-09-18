# D-06 — Bindings are immutable and declared once

- Status: accepted
- Rules: `BND-LET`, `BND-REDECLARE`, `BND-PARAMETER-SCOPE`
- Phase: 3
- Slice: P3.3 (resolve local bindings and capture environments)
- Migration: [Redeclaring a name](../MIGRATION.md#redeclaring-a-name)

## Context

`let` writes into a map, so declaring a name twice replaces the first binding
without comment. That is assignment wearing a declaration's syntax: it reads
like two independent bindings and behaves like one mutable variable.

```flux
let x = 1
let x = 2   // today: silently replaces
print(x)    // 2
```

## Decision

A binding is immutable, and a name may be declared once per scope. Declaring it
again in the same scope is rejected. Declaring it in an inner scope shadows the
outer one, which is a different name, not a new value for the same one.

## Consequences

This is a breaking change for any program that redeclares a name. It is also
what makes `BND-CAPTURE-IDENTITY` meaningful: if a name cannot be rebound, a
closure cannot see it change. Shadowing in an inner scope stays legal, so the
migration is usually to rename or to move the second declaration into a block.
