# D-04 — Statement boundaries

- Status: accepted
- Rules: `LEX-WHITESPACE`, `GRM-ADJACENCY`, `GRM-STATEMENT-SEPARATOR`
- Phase: implemented
- Slice: P3.1 (add precedence levels and checked value operations)
- Migration: none; the separator is optional

## Context

A newline is whitespace, so a statement ends where the grammar can no longer
continue it. That is unambiguous but occasionally surprising: a line beginning
with `(` continues the expression above it rather than starting a new one.

```flux
let f = fn(x) => x
f
(1)      // one call, not a name followed by a grouped 1
```

Automatic semicolon insertion would remove the surprise and introduce a larger
one: the rule that decides where a statement ends becomes invisible, and
reformatting a program can change what it does.

## Decision

Keep newlines as whitespace. Add an optional `;` that separates statements, for
the cases where juxtaposition reads as one expression. Flux does not insert it.

```flux
f; (1)   // phase 3: two statements, said so explicitly
```

## Consequences

`;` becomes a token, so `print(1); print(2)` stops being a lexical failure. No
program that parses today changes meaning, because today none contains a `;`.
