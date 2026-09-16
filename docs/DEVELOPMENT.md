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
  comparisons; operators within each level associate left to right.
- `parser/` configures Participle and returns the AST. Braces can represent a
  dictionary or an expression block; `{}` is an empty dictionary.
- `types/` maintains nested type environments. `UnknownType` is compatible with
  other types, but must not be used as an equality test for whether a type is
  unknown. Function annotations and concrete collection members are checked.
- `runtime/` evaluates AST nodes with a fresh global environment for every run.
  Closures capture enclosing function parameters; globals are resolved at call time.
- `compiler/` emits opcodes and four-byte unsigned operands. Each expression leaves
  exactly one value, including `nil` for void. Blocks discard intermediate results
  and conditional jumps consume their condition.
- `vm/` executes bytecode with separate local and global environments. Calls create
  another VM frame and preserve captured locals.
- `main.go` provides the CLI. Compilation embeds the current VM source alongside
  serialized bytecode in a temporary build directory, then invokes Go. Generated
  programs need neither the Flux checkout nor Go at runtime.
- `config/` loads `flux.json` from the working directory, overlaying defaults.
- `vsce/` contributes syntax highlighting and bracket/comment configuration; it
  has no language server or snippets.

## Validation

```sh
make check
make test-coverage
make build-all
```

`execution_test.go` checks the same programs against the interpreter and VM,
including all runnable examples, false booleans, operator precedence, escaped
strings, nested calls/blocks, closure capture, dictionary order, empty collections,
large bytecode operands, and runtime failures.

`main_test.go` builds the CLI and checks an actual standalone executable outside
the repository. It also covers configuration modes, error exit statuses, safe
initialization, version metadata, and cleanup of generated source files.

`types/types_test.go` covers strict, lenient, warn-only, and disabled modes;
`config/config_test.go` covers defaults, partial configuration and persistence.

`source/source_test.go` covers tabs, CRLF, empty files, EOF, multiline spans,
combining marks, and supplementary-plane characters; `diagnostic/diagnostic_test.go`
covers code groups, builder aliasing, ordering, deduplication, and suppression.
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
