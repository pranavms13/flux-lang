# D-07 — A captured name keeps its binding

- Status: accepted
- Rules: `BND-CAPTURE`, `BND-CAPTURE-IDENTITY`
- Phase: implemented
- Slice: P3.3 (resolve local bindings and capture environments)
- Migration: [Late name rebinding](../MIGRATION.md#late-name-rebinding)

## Context

A function literal captures the locals in scope where it is written, but a free
name that is not local is looked up in the global environment when the function
runs. Rebinding that name afterwards changes what an already-created function
sees.

```flux
let y = 1
let g = fn() => y
let y = 2
print(g())   // today: 2
```

Lexical resolution is a semantic decision, not an optimisation: the binding a
name refers to has to be decided before evaluation, or unrelated later
statements can change what a function means.

## Decision

Resolve every identifier to a declaration before evaluation. A captured name
keeps the binding it resolved to, for the whole life of the function.

## Consequences

Combined with [D-06](bindings.md), top-level rebinding disappears, so the change
is only observable through block-scoped shadowing. The resolver runs before both
engines, so the interpreter and the VM cannot drift on it.
