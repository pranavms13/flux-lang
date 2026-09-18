# D-16 — How values are displayed

- Status: accepted
- Rules: `EVL-DISPLAY-SCALAR`, `EVL-DISPLAY-CONTAINER`, `EVL-DISPLAY-OPAQUE`
- Phase: 3
- Slice: P3.1 (shared value semantics)
- Migration: [Displayed containers](../MIGRATION.md#displayed-containers)

## Context

Display is the host's default formatting. Scalars come out right by coincidence.
Containers come out in a shape no Flux program can write back:

```flux
print([1, 2])     // [1 2]
print({"a": 1})   // map[a:1]
```

A function is worse: it displays a host address, so the output differs between
runs, and the two engines print different text for the same value.

## Decision

Specify display, and implement it over Flux values rather than delegating:

- `int` shows decimal digits; `string` shows its characters unquoted; `bool`
  shows `true` or `false`, never `yes` or `no`.
- A list shows `[a, b, c]` and a dictionary `{k: v}`, with nested strings quoted
  so a displayed value can be told from a displayed name.
- A function shows a fixed placeholder, and so does a void value where one is
  displayed at all. Neither contains a host address, and both are identical on
  the two engines.

## Consequences

Any program or test that reads displayed container output changes. Scalar output
is unaffected, which is what the existing examples depend on. Once display is
Flux's own, `DIA-ENGINE-PARITY` extends to it: today the engines genuinely
disagree about what a function prints.
