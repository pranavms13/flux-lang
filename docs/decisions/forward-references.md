# D-08 — Forward references and self-recursion

- Status: accepted
- Rules: `BND-FORWARD-REFERENCE`, `BND-SELF-RECURSION`
- Phase: implemented
- Slice: P3.4 (introduce typed self-recursion)
- Migration: [Recursive functions](../MIGRATION.md#recursive-functions)

## Context

The checker binds a name after checking its value, so a function cannot refer to
the name it is being bound to. Recursion is therefore rejected — and then works
anyway if checking is relaxed, because the interpreter resolves the name when
the call happens rather than when the function was written.

```flux
let sum = fn(n: int): int => if n > 0 then n + sum(n - 1) else 0
print(sum(3))
// strict and lenient: rejected. warn-only and disabled: prints 6.
```

Two engines agreeing on an answer the checker rejects is the worst of both: the
feature exists, and the only way to use it is to turn off the checking.

## Decision

An ordinary binding must be declared before it is used. A function may refer to
itself by the name it is being bound to, provided a complete signature is supplied by the variable annotation or all
parameter and return annotations. The checker binds that signature before
checking the body, without solving recursive inference.

Mutual recursion between separate declarations stays unavailable; it needs a
declaration group, and there is no evidence yet that it is wanted.

## Consequences

The checker gains a pre-binding step for a `let` whose value is an annotated
function literal. The relaxed modes stop being the only way to write a recursive
program, which is the point.

Implemented in Phase 3 with a default limit of 256 active user calls and a
32-frame rendered trace cap. Binding errors are never downgraded by type modes.
