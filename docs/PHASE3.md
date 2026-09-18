# Phase 3 implementation and validation

Completed 2026-09-19, from `main` at
`bcbd0ed4db3c19e20906a182e0488ff5f39d030b`. The starting tree was clean and
`go test ./...` passed before implementation. Go remains pinned to a minimum of
1.23.2 and Participle remains at v2.1.4. No dependency was added.

Before completion, main was fetched again and fast-forwarded to `bf54906`
(the intervening GitHub Actions token-permission fix). It touched no Phase 3
implementation files; the working changes were preserved.

## Implementation

- P3.1: eight precedence levels, longest-match operator lexing, checked int64
  operations and literal magnitudes, structural comparable-value equality,
  stable scalar/container/function display, and bounded list-index conversion.
  `value/` is part of the closed standalone source bundle. Format 2 explicitly
  rejects earlier bytecode and carries int64 constants and capture metadata.
- P3.2: the interpreter skips the right operand when the left decides a logical
  result; the compiler emits consuming boolean branches with replacement
  constants. Both operands are still checked statically. All generated branches
  and boolean checks retain their source locations.
- P3.3: optional positioned semicolons and block statements, resolver binding IDs,
  declaration states, scopes and transitive captures. The resolver runs before
  optional checking. Both engines separate globals from locals and preserve
  captured cells after return; VM block exit restores the local scope while
  leaving one result.
- P3.4: complete recursive signatures from variable or parameter/return
  annotations, signature prebinding and body checking, and cells initialized
  before calls. Global and local recursive closures work in all execution paths.
  The default execution limit is 256 active calls; rendered traces show at most
  32 innermost frames. Mutual recursion remains deferred.

The promoted conformance fixtures run in all four checking modes on both engines.
`phase3_test.go` additionally covers numeric extrema and failures, operator
precedence, skipped side effects, nested stack usage, returned/transitive captures,
shadowing, signature errors, bounded recursion, serialization and more than 255
local binding IDs. Parser, resolver, value and VM unit tests cover positions,
identity metadata, malformed operands, jump boundaries and format rejection.
`examples/core.flux` demonstrates the features and belongs to the checked example
manifest. The VS Code grammar highlights every new operator and semicolons.

## Validation

On macOS ARM64 / Apple M3 Pro:

- `go test ./...` with Go 1.23.2, including standalone builds and execution.
- `go test -race -coverpkg=./... ./...` with Go 1.25.1. The root integration
  suite exercises 75.7% of all-package statements; this is coverage evidence,
  not a release threshold.
- `go vet ./...`, an assertion that `gofmt -s -l .` is empty, and
  `git diff --check`.
- `make build-all`: Linux AMD64/ARM64, macOS AMD64/ARM64 and Windows AMD64.
  Only native macOS ARM64 executables were run.
- Standalone conformance executables exercise global/local recursion, returned
  captures, short circuiting, display and runtime failures after the source file
  has been removed. The existing offline build-bundle and debug-on/off checks
  remain enabled.
- `go test ./value -run '^$' -fuzz FuzzArithmetic -fuzztime=15s`: 4,051,607 cases,
  using `math/big` as an independent oracle for `+`, `-`, `*`, `/`, `%` and
  their overflow/zero-divisor boundaries.

No release was published. Migration notes designate v1.0.0 as the intended
breaking release and describe bytecode format 2; local builds retain `dev`
version metadata unless build flags supply a release version.

## Performance measurements

Go 1.25.1 on the same machine, 100 ms benchmark windows:

| Benchmark | Time/op | Bytes/op | Allocations/op |
| --- | ---: | ---: | ---: |
| Parse small | 504 µs | 532,989 | 6,521 |
| Parse nested | 5.71 ms | 7,526,680 | 88,958 |
| Parse large | 13.2 ms | 10,438,133 | 119,538 |
| Resolve/check small | 11.2 µs | 9,717 | 103 |
| Resolve/check nested | 18.0 µs | 12,862 | 159 |
| Resolve/check large | 401 µs | 255,544 | 844 |

The pre-change smoke baseline used one benchmark iteration: parsing took
947 µs / 7.38 ms / 13.2 ms for small/nested/large. Single-iteration measurements
include warmup and are not comparable performance budgets. Historical sustained
measurements remain in [BASELINE.md](BASELINE.md).

Nested parsing remains exponential because a brace may start a dictionary or a
block. At depths 2/4/6/8 the measured times were 474 µs / 1.51 ms / 5.53 ms /
21.4 ms. The expanded grammar adds overhead; removing redundant expression
alternatives alone did not solve the ambiguity. No performance budget or parser
replacement is claimed. Constraint inference, formatting, editor analysis,
parser depth/work budgets and broader tooling remain later-phase work.
