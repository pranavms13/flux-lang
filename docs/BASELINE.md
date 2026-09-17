# Phase 1 implementation baseline

Recorded 2026-09-17, before the first Phase 1 slice in
[docs/PLAN.md](PLAN.md). This file states what "unchanged behavior" means for
the rest of the phase, so a later regression can be traced to a commit and a
toolchain rather than to a memory of how things used to work.

## Commit

Phase 1 branches from `503377e6c2573325b8ecbd8e43db922de798b3bc`
("fix: stabilize Flux execution and document development roadmap"), on a clean
working tree. The stabilization changes the plan describes as uncommitted are
part of that commit; there is no separate change set left to checkpoint.

Work happens on `feat/phase-1-diagnostics`.

## Toolchains

| Component | Version at baseline |
| --- | --- |
| Go (local) | 1.25.1 darwin/arm64 |
| Go (module requirement) | 1.23.2 — unchanged; new packages must build against it |
| Participle | v2.1.4, pinned |

`go.mod` keeps `go 1.23.2`. A newer local toolchain compiles the module, but no
Phase 1 slice may depend on a language or standard-library feature that 1.23.2
lacks. Generated executables shell out to whatever `go` is installed on the
user's machine, so this constraint applies to the build bundle as well.

## Test suite

`go test ./...` passes at the baseline commit. Packages with tests: the root
package (backend parity and examples), `config`, `lexer/tests`, and `types`.
`ast`, `compiler`, `internal/testutil`, `lexer`, `parser`, `runtime`, and `vm`
have no tests of their own.

`examples/type_errors.flux` is invalid on purpose: it is the fixture that proves
the checker rejects a bad program. It must keep failing. A Phase 1 change that
makes it pass is a regression in the checker, not a fixed example.

## Fixture manifest

`internal/fixtures` declares every file under `examples/`: what it proves, and
what it is expected to do in each of the four type-checking modes. A fixture's
kind is derived from those outcomes rather than declared, so the classification
cannot disagree with the behavior the tests check.

| Fixture | Kind | Strict | Lenient | Warn-only | Disabled |
| --- | --- | --- | --- | --- | --- |
| `main.flux` | valid | output | output | output | output |
| `typed.flux` | valid | output | output | output | output |
| `array.flux` | valid | output | output | output | output |
| `dict.flux` | valid | output | output | output | output |
| `type_errors.flux` | mode-dependent | static error | static error | runtime error | runtime error |
| `flux.strict.json` | config | selects strict | | | |
| `flux.lenient.json` | config | selects warn-only | | | |

Two entries show why guessing from a filename does not work. `type_errors.flux`
is not simply "an invalid program": downgrading its errors to warnings does not
make it run, it makes it fail later, because the mismatches the checker reports
are real. And `flux.lenient.json` selects warn-only, not the lenient mode that
shares its name.

`TestFixtureManifestCoversExamples` fails on any file in `examples/` that the
manifest does not declare, so a new fixture cannot be added without saying what
it is for.

## Benchmarks

`go test -bench . -benchmem`, Apple M3 Pro (darwin/arm64), Go 1.25.1, at the
baseline commit. Three shapes: `small` is an ordinary seven-line script,
`nested` is six levels of nested conditionals plus a closure, `large` is 300
bindings and a 300-element list.

| Stage | small | nested | large |
| --- | --- | --- | --- |
| Parse | 298 µs, 259 KB | 2.64 ms, 3.5 MB | 8.01 ms, 5.0 MB |
| Type check | 1.23 µs, 1.3 KB | 1.75 µs, 1.7 KB | 34.9 µs, 38 KB |
| Compile | 1.49 µs, 2.3 KB | 1.82 µs, 3.0 KB | 31.4 µs, 63 KB |
| Interpret | 3.82 µs, 1.5 KB | 2.50 µs, 1.6 KB | 36.9 µs, 47 KB |
| VM | 3.81 µs, 2.1 KB | 2.66 µs, 2.4 KB | 27.4 µs, 52 KB |

Two facts to design against:

**Parsing dominates by two to three orders of magnitude.** Parsing a 270-byte
script costs 298 µs and 259 KB, while checking it costs 1.2 µs and running it
costs 3.8 µs. Phase 1 adds positions and a retained token stream to the parser,
which is where any regression will be least visible in relative terms and most
expensive in absolute ones. Measure the parser specifically.

**Parsing nested conditionals costs exponential time.** With
`participle.UseLookahead(participle.MaxLookahead)`, each alternative of `Expr`
is re-tried over the whole nested subtree:

| Nesting depth | 2 | 4 | 6 | 8 |
| --- | --- | --- | --- | --- |
| Parse | 278 µs | 772 µs | 2.64 ms | 9.98 ms |
| Allocations | 3.6 k | 12 k | 47 k | 185 k |

Roughly 1.8x per level, and it does not stop: sixteen nested conditionals is a
450-byte program that takes over two seconds to parse. Group expressions and
operator chains are unaffected — the cost is specific to nesting the `Expr`
alternation. `BenchmarkParseNesting` records the curve. This constrains the
plan's intent to keep Participle: it is a grammar problem to fix in Phase 3
rather than a reason to replace the parser, but no Phase 1 grammar change should
make the curve worse, and fixture programs must stay shallow.

Both execution backends perform within a few percent of each other on all three
shapes, which is the parity Phase 1 must preserve while adding source maps.

## What P1.2 cost

Re-measured after source positions landed, on the same machine. Every AST node
now embeds two `lexer.Position` values, which Participle fills as it builds the
tree.

| Benchmark | Before | After | Change |
| --- | --- | --- | --- |
| Parse small | 298 µs, 259 KB | 351 µs, 354 KB | +18% time, +37% memory |
| Parse nested | 2.64 ms, 3.5 MB | 3.54 ms, 5.0 MB | +34% time, +43% memory |
| Parse large | 8.01 ms, 5.0 MB | 9.22 ms, 6.7 MB | +15% time, +34% memory |
| Type check large | 34.9 µs | 45.7 µs | +31% |
| Compile large | 31.4 µs | 45.0 µs | +43% |
| Interpret large | 36.9 µs | 48.7 µs | +32% |
| VM large | 27.4 µs | 27.7 µs | unchanged |

The parser regression is the price of the positions themselves. The regressions
in the checker, the compiler, and the interpreter are a consequence rather than
a second cost: each node grew by two positions, so walking the same tree touches
more memory. The VM is unchanged, which corroborates that reading — it executes
bytecode and never walks the AST.

Most of the per-node cost is the `Filename` string that `lexer.Position` carries
twice on every node, duplicating one value the source snapshot already holds.
Participle requires a field its `lexer.Position` converts to, and Go struct
conversion needs identical underlying types, so a compact position cannot be
injected directly. Shrinking this means a pass that rewrites positions into
`source.Span` after parsing, which is worth doing only if a measurement later
says it matters.

## What P1.3 cost

| Benchmark | Before | After |
| --- | --- | --- |
| Type check small | 1.21 µs, 17 allocs | 1.34 µs, 19 allocs |
| Type check nested | 1.67 µs, 22 allocs | 1.85 µs, 25 allocs |
| Type check large | 45.7 µs | 43.4 µs |

About 11% on the two function-heavy shapes, from the parameter provenance each
function literal now records so that an argument mismatch can point back at the
parameter it disagrees with. Building spans and diagnostics costs nothing on a
program that checks cleanly, because those paths only run when something is
wrong. The large shape is flat: it declares 300 bindings and one list, and no
functions.

## Still not recorded

No performance budgets are set. Set them against measurements taken after the
runtime work in P1.4, not against either column above.
