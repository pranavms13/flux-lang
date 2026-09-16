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

## Not yet recorded

- Parser, checker, and VM benchmarks for small, nested, and large programs.
  Performance budgets stay unset until those numbers exist.
- A fixture manifest separating valid programs, static errors, runtime errors,
  and mode-dependent programs. Until it exists, no test may infer a fixture's
  expected outcome from its filename.
