# D-14 — Collections

- Status: accepted
- Rules: `VAL-LIST-HOMOGENEOUS`, `VAL-DICT-HOMOGENEOUS`, `VAL-DICT-KEY-TYPE`, `VAL-DICT-DUPLICATE`, `VAL-DICT-MISSING`, `VAL-IMMUTABLE`
- Phase: implemented
- Slice: none for the core rules; library operations are P5.2
- Migration: none

## Context

Lists and dictionaries are homogeneous, immutable, and keyed by `int`, `string`
or `bool`. There are no operations on them beyond indexing, which is a gap in
the library rather than a gap in the language.

## Decision

Keep the core rules as they are, and state them:

- Every element of a list shares one type; every key and every value of a
  dictionary likewise.
- Keys are `int`, `string` or `bool`. A function or a container is not a key,
  which follows from [D-13](equality.md): a key must be comparable.
- A literal that repeats a key keeps the last pair, and evaluates both values.
- Reading an absent key is `R_MISSING_KEY`, not a default.
- Nothing modifies a collection in place, because there is no assignment.

```flux
let d = {"k": 1, "k": 2}
print(d["k"])   // 2
```

## Consequences

Phase 5 adds library functions that return new collections rather than mutating
them, which the immutability rule already requires.
