# D-15 — No implicit conversion

- Status: accepted
- Rules: `TYP-NO-COERCION`
- Phase: implemented
- Slice: none for the rule; conversion functions are P5.2
- Migration: none

## Context

`+` has two overloads, `int + int` and `string + string`, and mixing them is an
error in both directions. Nothing converts a number to a string to make an
expression work.

## Decision

Keep it. No operator converts its operands, in any mode, including the relaxed
ones: a lenient mode exists to defer reporting a mistake, not to change what a
program computes.

```flux
print(1 + "1")   // T_OPERAND_TYPE, not "11" and not 2
```

## Consequences

Turning a number into a string needs a function, which Phase 5 adds as part of
the standard library. Until then, a program that wants one has no way to write
it — an acceptable gap, and a visible one.
