# D-09 — Integers are 64-bit and checked

- Status: accepted
- Rules: `VAL-INT`, `VAL-INT-OVERFLOW`
- Phase: implemented
- Slice: P3.1 (add precedence levels and checked value operations)
- Migration: [Integer overflow](../MIGRATION.md#integer-overflow)

## Context

`int` is the host's `int`, so its width follows the build target, and arithmetic
past the limit wraps silently.

```flux
print(9223372036854775807 + 1)   // today: -9223372036854775808
```

Wrapping is a reasonable policy for a systems language and a poor one for a
language people are learning in: the wrong answer arrives without a word, and
the next thing that fails is somewhere else.

## Decision

`int` is a signed 64-bit integer on every target. An operation whose result does
not fit is reported as `R_INT_OVERFLOW` at the operator that produced it.

Fixed width and checked arithmetic are a Flux choice, not something the host
provides: Go's signed overflow is defined to wrap, so the operator helpers have
to test the boundary themselves.
[Host arithmetic reference](https://go.dev/ref/spec#Arithmetic_operators).

## Consequences

AST literals, bytecode constants and runtime values all move from `int` to
`int64`, which is why this is a slice of its own rather than a patch to the
operators. A program that relied on wrapping changes behavior; a program that
never overflowed does not.
