# D-01 — Top-level expression output

- Status: accepted
- Rules: `EVL-TOP-LEVEL-DISPLAY`, `EVL-PRINT`
- Phase: implemented
- Slice: none; this records existing behavior
- Migration: none

## Context

A top-level expression whose value is not void is displayed. `print(v)` writes a
line as a side effect and produces void. The two overlap: `print(1)` as a
statement displays `1` once, not twice, because the call's own value is void.

Removing automatic display would be the tidier language, and it would break
every script that relies on it, including three of the five examples in this
repository. Keeping it costs one rule.

## Decision

Preserve automatic display of non-void top-level expression values. Keep `print`
as an ordinary function whose effect is immediate and whose value is void.

```flux
let x = 1   // displays nothing: a declaration is not an expression
x           // displays 1
print("p")  // displays p once; the call's void value is not displayed
```

## Consequences

Void must stay undisplayable, or every `print` statement would print a second
line. That is why `EVL-DISPLAY-OPAQUE` specifies a placeholder for void only
where a void value is passed somewhere that displays it explicitly.
