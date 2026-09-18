# D-03 — What `{}` means

- Status: accepted
- Rules: `GRM-EMPTY-BRACES`
- Phase: implemented
- Slice: none; this records existing behavior
- Migration: none

## Context

`{}` is ambiguous between an empty dictionary and an empty block. The grammar
resolves it as a dictionary, because the dictionary alternative is tried first
where the two overlap.

Either reading is defensible. What is not defensible is leaving it undecided:
the two differ in type and in what indexing them reports, so a program that
relies on the wrong one fails in a way the message does not explain.

## Decision

`{}` is the empty dictionary. A block that produces void is written with an
expression in it whose value is void.

```flux
{}["a"]  // R_MISSING_KEY: it is a dictionary with no such key,
         // not a void block that cannot be indexed at all
```

## Consequences

There is no literal for an empty block, and no need for one: a block exists to
sequence expressions, and an empty sequence has nothing to sequence.
