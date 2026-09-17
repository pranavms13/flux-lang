# Development baseline

Flux is a small Go language implementation with an optional static checking pass,
a tree interpreter, a bytecode compiler/VM, and a declarative VS Code grammar.
There are no external services or persistent application databases.

## Execution paths

```text
source -> lexer -> Participle parser -> AST -> optional type checker
                                              |              |
                                          flux run       flux compile
                                              |              |
                                         interpreter     bytecode compiler
                                                             |
                                                   gob-encoded Chunk + VM
                                                             |
                                                       Go executable
```

- `source/` holds immutable source snapshots and the half-open byte spans that
  locate constructs in them. `Position` derives line and column numbers in four
  units — bytes, runes, UTF-16 code units, and tab-expanded display cells —
  because a byte offset is none of those.
- `diagnostic/` describes reported problems as data: code, severity, message,
  primary span, related locations, notes, help. A `Bag` orders them
  deterministically and withholds the follow-on errors that one root failure
  causes. `ToolError` is the separate, source-less kind for a missing file or a
  missing Go toolchain. Neither package imports anything else in the module, so
  the standalone build bundle can include them beside the VM.
- `ast/` defines the parser grammar. Addition/subtraction bind more tightly than
  comparisons; operators within each level associate left to right. Every node
  embeds `ast.Node`, which Participle fills with the node's start and end
  positions; `Span` converts that pair into a `source.Span`, and returns
  `NoSpan` for a node built by hand rather than parsed.
- `parser/` configures Participle and returns a `Result`. `ParseSource` takes a
  source snapshot and reports failures as diagnostics located in it; `Parse`
  remains for callers that only have text. A failed parse leaves `Program` nil
  and puts the recoverable tree in `Partial`, which must never be executed. The
  full token stream, comments and whitespace included, is captured once on
  `ast.Program`; text after its `EndPos` is trivia and comes from the snapshot.
  Braces can represent a dictionary or an expression block; `{}` is an empty
  dictionary.
- `types/` maintains nested type environments. `UnknownType` is compatible with
  other types, but must not be used as an equality test for whether a type is
  unknown. Function annotations and concrete collection members are checked.
  Diagnostics are records, not strings: `NewTypeCheckerForSource` locates them,
  a mismatch points at the offending construct with the declaration it
  disagrees with attached, and `report` is the only place strict and warn-only
  change a severity. `FunctionType.Params` carries where each parameter was
  written, which is provenance rather than part of the type, so `Equals`
  ignores it.
- `fault/` is the catalogue of failures a program can produce while it runs.
  Both engines report through it, so a failure has one code and one wording
  whichever engine noticed it, and `fault.Report` renders it for a terminal —
  including in a generated executable, which has no CLI to do it.
- `runtime/` evaluates AST nodes with a fresh global environment for every run.
  Closures capture enclosing function parameters; globals are resolved at call time.
  `Run` returns failures rather than panicking and takes its output writer
  through `Options`, so a test reads what a program printed without replacing a
  global.
- `compiler/` emits opcodes and four-byte unsigned operands. Each expression leaves
  exactly one value, including `nil` for void. Blocks discard intermediate results
  and conditional jumps consume their condition. `NewFluxCompilerForSource` records
  a source map on every chunk, keyed by each instruction's own start offset and
  already resolved to file, line and column, so a built executable can still
  report a position once its source is gone.
- `vm/` executes bytecode with separate local and global environments. Calls create
  another VM frame and preserve captured locals. `Run` returns failures; the
  instruction offset is captured before operands are read, so a failure names the
  operation that failed rather than the next one. A stack shortfall is reported as
  an internal defect, because only a compiler bug can cause one.
- `main.go` provides the CLI. Compilation writes the packages listed in
  `bundledPackages` into a temporary module alongside the generated program, then
  invokes Go with `GOWORK=off` and `GOPROXY=off`. Every bundled package depends
  only on the standard library and on other members of the bundle, which
  `TestBuildBundleIsClosed` enforces. Generated programs need neither the Flux
  checkout nor Go at runtime.
- `config/` loads `flux.json` from the working directory, overlaying defaults.
- `vsce/` contributes syntax highlighting and bracket/comment configuration; it
  has no language server or snippets.

## Validation

```sh
make check
make test-coverage
make build-all
```

`execution_test.go` checks that both engines agree on a failure's code, message
and location, not merely that both refuse to finish, and that a failure inside a
function carries the call sites that led there. It checks the same programs
against the interpreter and VM,
including all runnable examples, false booleans, operator precedence, escaped
strings, nested calls/blocks, closure capture, dictionary order, empty collections,
large bytecode operands, and runtime failures.

`main_test.go` builds the CLI and checks an actual standalone executable outside
the repository. It also covers configuration modes, error exit statuses, safe
initialization, version metadata, and cleanup of generated source files.

`types/types_test.go` covers strict, lenient, warn-only, and disabled modes;
`config/config_test.go` covers defaults, partial configuration and persistence.

`types/diagnostics_test.go` checks what each kind of diagnostic points at and
what it attaches, and that a checking mode changes severity without changing the
code or the location.

`parser/parser_test.go` checks the span of every kind of construct against the
text it claims to cover, pins Participle's end-position semantics, and covers
lexical, syntax, and end-of-input failures. `TestOnlyTheRootCapturesTokens` fails
if a node other than `Program` declares a `Tokens` field, which would copy each
subtree's tokens once per level of nesting.

`source/source_test.go` covers tabs, CRLF, empty files, EOF, multiline spans,
combining marks, and supplementary-plane characters; `diagnostic/diagnostic_test.go`
covers code groups, builder aliasing, ordering, deduplication, and suppression.

`internal/fixtures` declares what every file under `examples/` is for and what it
is expected to do in each type-checking mode; `fixtures_test.go` runs each one in
all four modes on both backends and fails on any example the manifest does not
declare. A fixture's kind is derived from its outcomes, never from its filename.

`bench_test.go` records the parser, checker, compiler, interpreter and VM
baseline; see [BASELINE.md](BASELINE.md) for the numbers. Parsing dominates the
pipeline, and parsing nested conditionals costs exponential time, so keep test
and example programs shallow.
The existing lexer tests remain in `lexer/tests/`.

When adding language syntax, update the AST, checker, interpreter, compiler/VM,
editor grammar, documentation, and parity tests together. The compiler uses the
VM's Go source directly for standalone builds, so keep that file self-contained
or extend the embedding/build step if the VM is split into multiple files.

## Current semantics and limits

- `print(value)` emits one line immediately and returns void. Non-void top-level
  expression statements also display their result; block expressions return their
  last value without implicitly displaying intermediate values.
- Arithmetic supports `+` and `-`; comparisons support `==`, `<`, and `>`.
  Parentheses control grouping. Multiplication, division, unary negation, logical
  operators, loops, assignment, imports and explicit `return` are not implemented.
- Lists and dictionaries are homogeneous when checking is enabled. Dictionary
  keys are integers, strings, or booleans. Duplicate keys keep the last value.
- Lenient mode permits truthy conditions, differing branch types and unlike-type
  equality with warnings. It does not perform implicit string/number conversion.
  Warn-only and disabled modes may still encounter runtime errors.
- Inference is intentionally incomplete. Untyped parameters use `UnknownType`;
  it is not a constraint solver. Strict mode is strongest with explicit function
  signatures. Recursive/forward function references are not resolved by the checker.
- Compiler `optimizationLevel` and `debug` settings are reserved metadata and do
  not currently change generated code.
- Runtime diagnostics are readable CLI errors, but most lack source spans. The
  internal interpreter/VM APIs still signal runtime failures with panics.
- Bytecode is an internal format without a compatibility promise; rebuild programs
  after compiler changes. Previously built standalone executables are unaffected.

## Next development priorities

1. Add source positions and structured diagnostics throughout the AST and runtime.
2. Define stronger inference and recursive function checking, including the exact
   meaning of strict mode for unannotated functions.
3. Choose and specify the next language features, then implement them in both
   backends with shared behavioral tests.
4. Exercise release automation in GitHub. The workflow now accepts explicitly
   created release versions, grants release write permissions, fetches history for
   changelogs, and lets the release action create the tag. Remote publishing still
   needs validation; no release was created during this local review.
5. Add editor diagnostics/completion only after stable source-aware diagnostics exist.

## Verified in this review

- Tests pass on Go 1.23.2 and Go 1.25.1 on macOS ARM64.
- Race-enabled tests, `go vet`, formatting, and whitespace checks pass.
- All runnable examples pass through both the interpreter and VM; CLI integration
  tests build and execute actual standalone programs outside the checkout.
- Linux AMD64/ARM64, macOS AMD64/ARM64, and Windows AMD64 binaries cross-compile.
  Non-native executables were not run.
- Documentation generation, installation into a temporary bin directory, and
  repeated distribution packaging were verified.
- VS Code manifest/grammar/configuration JSON and referenced paths were checked.
  An interactive extension-host session and remote GitHub publishing were not run.
