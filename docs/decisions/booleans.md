# D-12 — Booleans and conditions

- Status: accepted
- Rules: `VAL-LOGICAL`, `EVL-IF-TRUTHY`, `EVL-IF-BOOL`
- Phase: 3
- Slice: P3.2 (implement short-circuit control flow)
- Migration: [Truthy conditions](../MIGRATION.md#truthy-conditions)

## Context

There are no logical operators: `!`, `&&` and `||` are lexed and appear in no
rule. Separately, a condition that is not a `bool` is accepted outside strict
mode and read for truthiness — `0` and `""` are false, everything else is true.

The two are tangled in practice. Truthiness exists because there is no way to
write "is this list empty"; adding the operators removes most of the reason for
it, and leaving it in place would mean `if xs` and `if count > 0` quietly mean
different things.

## Decision

Add `!`, `&&` and `||`. They take `bool` and produce `bool`. `&&` and `||` do
not evaluate their right operand when the left one decides the answer.

Separately, require a condition to be a `bool` in every mode. Truthiness is
removed rather than downgraded, so there is one rule instead of a rule per mode.
The checker reports `T_CONDITION_TYPE`; when checking is relaxed or disabled,
runtime validation reports `R_CONDITION_TYPE` at the condition.

```flux
if count > 0 && ready then "go" else "wait"
```

## Consequences

Short-circuit evaluation needs the jump opcodes the VM already has
(`OpJumpIfFalse`, `OpJumpIfTrue`), so the compiler change is small and the
interpreter change is not. Removing truthiness breaks programs that relied on
`if 0`; they become `if n == 0`, which is what they meant.
