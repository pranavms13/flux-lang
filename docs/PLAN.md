# Flux language development plan

Research completed: 2026-09-17. Status: proposed implementation roadmap.

This plan develops the five agreed areas in order: diagnostics, language semantics,
core expressions/functions, type inference, and standard library/tooling. The
research below was completed before this file was created. Checkboxes describe
future work, not functionality already implemented.

The baseline is the current working tree, including the stabilization changes
from the project review. Those changes are still uncommitted. Existing behavior
is summarized in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). This roadmap does not
assume a clean checkout, a published release, or a new dependency version.

## 1. Intended outcome and boundaries

Flux should become a small, predictable language for scripts and learning, with
useful diagnostics and matching interpreted/compiled behavior. This is a working
product direction, not a claim that the language already supports general-purpose
application development.

By the end of this plan, a developer should be able to:

- Understand an error from its source location, explanation, and call trace.
- Consult a specification that explains scope, operators, values, and type modes.
- Use ordinary arithmetic, boolean expressions, local bindings, and recursion.
- Rely on inferred relationships between function inputs and outputs.
- Check and format programs, use a small standard library, and receive editor
  diagnostics, completion, hover information, and definition navigation.

Out of scope for these phases: modules/package registries, networking and file-I/O
libraries, mutation, loops, classes, user-defined algebraic data types, exceptions
as a language feature, concurrency, JIT/native-code generation, optimization
passes, and a debugger. These deserve separate designs after this foundation.
Go remains the implementation language; Participle remains the parser unless a
measured limitation justifies replacing it.

## 2. Current architecture and constraints

| Area | Current state | Consequence for this plan |
| --- | --- | --- |
| Front end | Participle v2.1.4; grammar lives in `ast/`; parser uses `<stdin>` as its filename | Preserve the dependency initially; add named source input and AST positions |
| Grammar | Addition/subtraction, one comparison level, calls/indexing, expression-only blocks | Introduce explicit precedence levels and statements in blocks |
| Type checker | Nested environments, string diagnostics, broadly compatible `UnknownType` | Separate diagnostic data from rendering, then replace inference placeholders |
| Interpreter | Fresh globals per run, captured parameter maps, runtime panics | Add returned errors, injected output, and resolved lexical bindings |
| Compiler/VM | Four-byte operands; one result per expression; jumps consume their condition | Preserve stack invariants and account for consuming jumps in short-circuit code |
| Standalone build | `main.go` embeds only `vm/vm.go`, rewrites its package, serializes `vm.Chunk` with gob | Shared runtime packages require a deliberate build-bundle change |
| CLI/config | `run`, `compile`, `init`, `version`; configuration comes from working directory | Extract reusable analysis; add checking/formatting/server commands later |
| Editor | Declarative TextMate grammar and language configuration | A language client and server are new components |
| Tests | Backend parity, examples, CLI binaries, type modes, config, lexer | Extend these fixtures; do not discard the stabilization regressions |

Review these files before each affected implementation: [AST](ast/datastructures.go),
[parser](parser/parser.go), [checker](types/types.go), [interpreter](runtime/runtime.go),
[compiler](compiler/compiler.go), [VM](vm/vm.go), [CLI](main.go),
[execution tests](execution_test.go), and [CLI tests](main_test.go).

## 3. Research findings and their application

The external sources establish techniques and protocol requirements. The proposed
Flux policies, package boundaries, and acceptance criteria below are engineering
recommendations for this repository, not rules imposed by those sources.

| Finding | Application to Flux | Primary source |
| --- | --- | --- |
| The pinned parser can populate `Pos`, `EndPos`, and captured tokens, including elided tokens | Use its existing position support and preserve a token stream for future formatting | [Participle v2.1.4: error reporting and comments](https://raw.githubusercontent.com/alecthomas/participle/v2.1.4/README.md) |
| Diagnostics can carry codes, primary/secondary locations, notes, and suggestions independently of display | Produce structured diagnostics once; provide terminal, JSON, and editor adapters | [Rust compiler diagnostic guide](https://rustc-dev-guide.rust-lang.org/diagnostics.html) |
| Lexical binding is resolved from source structure; later declarations must not accidentally change an existing closure's binding | Add a resolver shared by both execution paths | [Crafting Interpreters: resolving and binding](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/resolving-and-binding.md) |
| Closures need an explicit representation of captured bindings | Use binding identities and captured cells; account for local self-recursion | [Crafting Interpreters: closures](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/closures.md) |
| Short-circuit operators are control flow | Compile branches and verify both paths leave one result | [Crafting Interpreters: jumping back and forth](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/jumping-back-and-forth.md) |
| Host arithmetic has specific overflow and division behavior | Specify Flux arithmetic explicitly and implement checked operations | [Go specification: arithmetic](https://go.dev/ref/spec#Arithmetic_operators) |
| Inference can derive constraints and solve them with unification | Replace permissive unknowns with identifiable variables and a solver | [UMass type inference lecture](https://people.cs.umass.edu/~arjun/courses/compsci631-fall2017/reading/lecture8.pdf) |
| Let-polymorphism needs generalization and fresh instantiation; constraints need origins for useful errors | Introduce type schemes after basic inference, retaining source provenance | [Cornell: type inference](https://cs3110.github.io/textbook/chapters/interp/inference.html) |
| Mutation complicates polymorphic generalization | Keep collections/bindings immutable in this roadmap; revisit soundness before adding mutation | [OCaml manual: polymorphism and limitations](https://ocaml.org/manual/5.3/polymorphism.html) |
| Fuzz targets should be deterministic and avoid persistent global state | Inject output and use bounded, reproducible parser/evaluator targets | [Go fuzzing documentation](https://go.dev/doc/security/fuzz/) |
| Comment placement is difficult; formatter correctness needs explicit checks | Preserve comments, require idempotence and syntax/behavior preservation | [Prettier rationale](https://prettier.io/docs/rationale.html#comments), [correctness checking](https://prettier.io/docs/cli.html#--debug-check) |
| LSP positions have negotiated encoding; UTF-16 support is required by the selected 3.17 baseline | Convert canonical byte spans at the protocol boundary | [LSP 3.17 position definition](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/types/position.md) |
| Editor diagnostics and document versions are protocol data, separate from execution | Analyze in-memory document snapshots and reject stale results | [LSP diagnostics](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/types/diagnostic.md), [document changes](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/textDocument/didChange.md) |
| A language server can be separate from its VS Code client | Keep language analysis in Go; add a thin TypeScript client | [VS Code language server guide](https://code.visualstudio.com/api/language-extensions/language-server-extension-guide) |

Version choices: retain Go 1.23.2 compatibility initially; use LSP 3.17 as a
specified implementation target, without claiming it is the newest protocol.
Evaluate and pin any future LSP/client dependency when that phase starts. Do not
copy the dependency versions from a moving documentation example without checking
compatibility with `vsce/package.json` and the project's Go version.

## 4. Phase order and delivery rules

| Phase | Deliverable | Prerequisite | Relative size |
| --- | --- | --- | --- |
| 1 | Source-aware diagnostics across interpreter, VM, and standalone binaries | Current stabilization baseline | Large |
| 2 | Executable language specification and recorded semantic decisions | Phase 1 diagnostic vocabulary | Medium |
| 3 | Complete core expressions, local bindings, lexical closures, typed recursion | Phase 2 decisions | Large |
| 4 | Constraint inference, type schemes, precise checking-mode rules | Phase 3 binding model | Large |
| 5 | Small standard library, `check`, formatter, language server and VS Code client | Phases 1–4 | Extra large; split into releases |

These sizes express implementation uncertainty, not calendar commitments. Re-size
work after the source-position and inference prototypes. Phase 5 has independent
sub-deliverables; do not wait for editor completion to ship the checker CLI.

Delivery rules for every implementation slice:

- Add a behavioral regression that fails for the missing capability.
- Implement the front end, checker, and both execution paths where applicable.
- Keep successful existing fixtures unless an explicit semantic decision changes them.
- Check diagnostic code/location as well as output and exit status.
- Update the specification, examples, grammar, and migration notes in the same slice.
- Keep source-checkout-independent executable builds working throughout.
- Keep new tests deterministic; record relevant performance baselines before setting budgets.
- Use the next phase only after its prerequisite acceptance criteria pass.

### Baseline preparation

- [x] Review and checkpoint the existing stabilization changes as their own change set.
      They landed in `503377e`; see [docs/BASELINE.md](BASELINE.md).
- [x] Record the exact commit and local toolchains used for the next implementation.
- [x] Run the existing suite and retain the intentionally invalid `type_errors.flux` fixture.
- [x] Record parser/checker/VM benchmarks for representative small, nested, and large programs.
      `bench_test.go`; numbers in [docs/BASELINE.md](BASELINE.md). Parsing dominates the
      pipeline by two to three orders of magnitude, and parsing nested conditionals costs
      exponential time, which `BenchmarkParseNesting` records.
- [x] Add a fixture manifest that distinguishes valid programs, static errors, runtime errors,
      and mode-dependent programs; never classify an example by filename alone.
      `internal/fixtures`; a fixture's kind is derived from its declared per-mode outcomes,
      and a file in `examples/` that the manifest does not declare fails the suite.

## 5. Phase 1 — Source positions and structured diagnostics

### Outcome

A language error identifies the actual file and expression. Interpreter execution,
VM execution, and generated executables report the same error category and source
location. Formatting a diagnostic does not depend on the component that found it.

### P1.1 — Establish source and diagnostic data models

Proposed packages: `source/` and `diagnostic/`. Keep these independent of the AST,
Participle, type checker, and execution engines so the standalone build can include
them without pulling in the front end.

- [x] Define a source identifier and immutable source snapshot with original bytes,
      display filename, and line-start index. `source.SourceID`, `source.Source`, and the
      `source.Map` registry, which is safe for concurrent use.
- [x] Use half-open byte spans `[start, end)` as the canonical location representation.
      `source.Span`, with `Union` for composite expressions and `Compare` for ordering.
- [x] Define conversions for terminal line/column and, later, negotiated LSP character units.
      `source.Position` counts bytes, runes, UTF-16 code units, and tab-expanded display
      cells separately; every field is 1-based, and the LSP adapter subtracts one.
- [x] Define diagnostics with a stable code, severity, message, primary span, optional
      related locations, notes, and optional help. Keep terminal styling out of this model.
- [x] Reserve code groups for lexing/parsing, binding, typing, runtime, and internal failures.
      `S_`, `B_`, `T_`, `R_`, and `X_`. `diagnostic.Register` declares a code with a
      description and rejects an unknown group or a duplicate at init time.
- [x] Distinguish a source-less tool error, such as an unreadable file, from a language error.
      `diagnostic.ToolError` carries no span and is a distinct type.
- [x] Define deterministic ordering and suppression of downstream errors caused by one root failure.
      `diagnostic.Bag` deduplicates identical reports, orders by position then code then
      arrival, and withholds diagnostics recorded with `AddCausedBy`.

Suggested API shapes, to refine in implementation:

```go
type Span struct {
    SourceID SourceID
    Start    int // byte offset, inclusive
    End      int // byte offset, exclusive
}

type Diagnostic struct {
    Code     string
    Severity Severity
    Message  string
    Primary  source.Span
    Related  []Label
    Notes    []string
}
```

Names and fields above are proposals, not existing APIs. Use Unicode strings and
comments in tests even while identifiers remain ASCII. Cover tabs, CRLF, empty
files, EOF, multiline spans, combining marks, and supplementary-plane characters.
Define terminal tab expansion explicitly; do not equate a byte offset with a
terminal display column or LSP UTF-16 column.

Implemented in `source/` and `diagnostic/`. `source/` has no internal dependencies,
and `diagnostic/` depends only on `source/`, so the standalone build bundle can
include both. Tab expansion is explicit: `source.DefaultTabWidth` is 8 and
`PositionWithTabWidth` accepts another stop width. Display columns count one
cell per non-tab rune and so do
not account for double-width or zero-width characters; that limitation is
documented on the field rather than hidden.

### P1.2 — Preserve positions through parsing

- [x] Add `ParseSource(filename, text)` or a source-snapshot equivalent; retain `Parse(text)`
      as a compatibility wrapper for tests and embedded callers. `parser.ParseSource` takes a
      `*source.Source`; `parser.Parse` remains, and `main.go` now reports the real filename.
- [x] Populate positions on statements, declarations, annotations, parameters, terms,
      operators, call arguments, indexes, and composite expressions. Every node embeds
      `ast.Node`, which Participle fills through the embedded struct.
- [x] Convert Participle positions at one boundary and verify end-position semantics
      against the pinned version with token/EOF fixtures. `ast.Node.Span` is the only
      conversion; `TestEndPositionSemantics` pins `EndPos` as the exclusive end.
- [x] Extract lexical and syntax failures through their typed errors, not string matching.
      `*lexer.Error` and `*participle.UnexpectedTokenError` via `errors.As`. The expected-set
      text is still recovered from the message, because Participle keeps it unexported.
- [x] Preserve a full file token stream once, including comments/trivia. Avoid copying
      the entire nested token range into every AST node. `Tokens` is declared on `ast.Program`
      alone, and `TestOnlyTheRootCapturesTokens` fails if another node declares one.
- [x] Define the result of a failed parse explicitly: diagnostics and optionally a partial
      syntax tree for tools; an erroneous tree must never be executed or compiled.
      `parser.Result.Program` is nil whenever parsing failed; the partial tree is in a
      separate `Partial` field that nothing executes.

Participle's built-in position/token capture makes a parser replacement unnecessary
for this step. Its partial-tree behavior still needs a Flux wrapper with a clear
execution boundary. [Pinned parser documentation](https://raw.githubusercontent.com/alecthomas/participle/v2.1.4/README.md).

Confirmed against the pinned version. Participle injects `Pos`/`EndPos` through an
embedded struct, sets `EndPos` from the next raw token (so it is the exclusive
end), and fills a `Tokens` field from that node's whole raw range, including
elided tokens. The stream stops at the last consumed token; whatever follows is
trivia by definition and is recovered from the source snapshot, so
`parser.Result.TrailingTrivia` needs no second lexing pass. Positions cost
roughly a sixth of parse time and a third of parse allocations; see
[docs/BASELINE.md](BASELINE.md).

### P1.3 — Migrate type diagnostics

- [x] Replace `errors []string` and `warnings []string` with diagnostic records.
      The checker holds a `diagnostic.Bag`.
- [x] Pass the relevant node/span into each type-checking operation. `checkOperator`,
      `CheckCallExpr`, `CheckIndexExpr` and `checkDictionaryKey` take spans; the rest
      reach their node directly.
- [x] Point an argument mismatch at the argument; attach the parameter declaration as
      related information. Do likewise for annotations and differing branch types.
      `FunctionType.Params` carries parameter provenance so the label can be built; it is
      empty for built-ins and for function types written as annotations, and the label is
      then omitted rather than guessed.
- [x] Apply strict/warn-only policy to severity at one layer while preserving codes and spans.
      `TypeChecker.report` is the only place a mode changes a severity, and
      `TestModesChangeSeverityAndNothingElse` checks that codes and spans are mode-independent.
- [x] Keep temporary string accessors only as adapters while migrating existing callers/tests.
      `GetErrors`/`GetWarnings` render from the bag and now include a position when the
      checker was given a source.
- [x] Do not introduce new inference behavior in this slice. The lenient/strict divergence in
      what a mismatched conditional returns is preserved as-is for Phase 4.

### P1.4 — Return runtime failures and carry VM source maps

- [x] Change interpreter evaluation and VM execution to return explicit errors/results.
      `runtime.Run` returns an error; `vm.Run` returns an error.
- [x] Replace ordinary language panics and unchecked assertions with validated operations.
      The VM's `need` checks stack depth before an instruction consumes it, and constant
      indexes and types are validated; a shortfall is an internal defect, not a user error.
- [x] Handle wrong arity, invalid operand types, missing keys, bad indexes, non-callables,
      and undefined values through the same runtime error catalogue. `fault/`, imported by
      both engines, so the two cannot drift apart on wording or codes.
- [x] Represent call traces as data containing function labels and call-site locations.
      `fault.Frame`, added as the failure unwinds so the innermost frame comes first.
- [x] Add a source map keyed by instruction-start byte offset to every compiled chunk,
      including nested function chunks. `vm.Chunk.Locations`; a nested function gets its
      own map, so a failure inside it reports its own line rather than the call's.
- [x] Capture the instruction start before reading operands, so failures point at the
      failing operation instead of a subsequent instruction.
- [x] Preserve function identity/location metadata when constructing closures. A chunk
      carries the name the function was bound to; a closure keeps its chunk.
- [x] Keep unexpected implementation failures distinguishable from expected user errors;
      CLI recovery must not disguise an internal defect as a type mismatch. Internal
      failures carry `X_INTERNAL`, and the CLI's recover now reports a panic as a Flux bug.
- [x] Inject an `io.Writer` through execution options. Replace global `os.Stdout` swapping
      in new tests and progressively migrate `internal/testutil/output.go` callers.
      Every caller was migrated, so `internal/testutil` is removed rather than left dead.

### P1.5 — Preserve standalone builds while sharing diagnostics

Adding a `diagnostic` import to `vm/vm.go` would break today's single-file packaging.
Resolve that dependency before merging the runtime API change.

Done ahead of schedule, because P1.4 could not land without it: the runtime API
change required the VM to import `fault`, and the old packaging only worked while
the VM imported nothing.

- [x] Replace package-name string rewriting with a small embedded source bundle containing
      the VM and its explicitly listed runtime-only dependencies. `bundledPackages` in
      `main.go` lists `diagnostic`, `fault`, `source`, `vm`.
- [x] Materialize those packages and a minimal temporary `go.mod` under the same module
      path; generate a main package that imports the bundled VM normally.
- [x] Exclude tests, parser/compiler sources, local workspace replacements, and accidental
      dependencies from the bundle. Keep bundle membership explicit and tested.
      `TestBuildBundleIsClosed` parses every bundled file's imports and rejects one that
      names a package outside the bundle or any external module.
- [x] Build with workspace discovery disabled; verify dependency resolution with
      `GOWORK=off` and `GOPROXY=off` using an already installed Go toolchain.
- [x] Give serialized chunks a format version and a deliberate gob registration scheme.
      `vm.Program` wraps the chunk with `vm.ProgramFormatVersion`; `vm.Encode`/`vm.Decode`
      own the registration, so the compiler and every generated executable use one scheme
      and a mismatched format is refused with an instruction rather than a decode error.
- [x] Store display filenames and precomputed line/column locations needed after the
      original source is removed. Never rely on an absolute development-machine path.
      Done in P1.4: `source.Location` holds the display filename with the line and column
      already resolved, and the CLI test deletes the source before running the executable.
- [x] Define `compiler.debug` to optionally embed source text for snippets; retain useful
      codes, locations, and traces when source text is omitted. Document size/source disclosure.
      `vm.Program.Sources`, filled only when `compiler.debug` is set. Without it the
      executable still reports code, position, and trace; `TestCLI` asserts it embeds no
      source, and `TestCompileDebugEmbedsSource` asserts the snippet comes from inside the
      binary by deleting the file first.
- [x] Verify cleanup on compilation failure and successful execution from another directory.
      `TestCompilationFailureCleansUp` counts leftover build directories around a failed
      compile; `TestCLI` runs the built executable from a different directory.

### P1.6 — Render and integrate

- [x] Add a plain-text renderer with optional terminal styling and stable no-color output.
      `render.Renderer`. Colour is off unless the destination is a terminal and always off
      under `NO_COLOR`; a test asserts that styling changes no text, only how it is drawn.
      Snippets expand tabs before drawing and measure the underline in the same columns.
- [x] Add a versioned JSON diagnostic representation for later CLI/editor reuse.
      `render.JSONVersion`; positions carry byte offsets and line/column together, because
      neither can be derived from the other without the source.
- [x] Keep diagnostics on stderr for execution; program output remains on stdout.
      `TestCLI` captures the two streams separately and asserts each is what it should be.
- [x] Define command exit behavior once and use it in the CLI and generated executable.
      `diagnostic.ExitSuccess`/`ExitFailure`/`ExitToolFailure`. An internal defect exits 2,
      not 1: it is not a failure of the user's program.
- [x] Add examples of diagnostics to the README and a diagnostic-code reference.
      [docs/DIAGNOSTICS.md](DIAGNOSTICS.md) is generated from the code registry, and a test
      fails when it drifts.

Target presentation, using illustrative wording and codes:

```text
main.flux:4:5: error[T_ARGUMENT_TYPE]: expected int, found string
4 | add("5", 10)
  |     ^^^
  = note: parameter 'a' is declared as int at main.flux:1:14
```

### Phase 1 completion criteria

- [x] Lexer, parser, checker, and ordinary runtime failures contain stable codes and locations.
      Thirty codes across five groups, listed in [docs/DIAGNOSTICS.md](DIAGNOSTICS.md).
- [x] Both backends and an actual generated executable agree on runtime error code and location.
      `TestRuntimeFailures` compares the two engines' codes, messages and spans rather than
      their prose; `TestCLI` checks the built executable reports what the interpreter did.
- [x] A generated executable still reports a location after its source file is deleted.
      `TestCLI` deletes it first. With `compiler.debug` it prints the line too, which
      `TestCompileDebugEmbedsSource` proves comes from inside the binary.
- [x] Human and JSON outputs represent the same diagnostic data.
      `TestTextAndJSONDescribeTheSameDiagnostic` renders one record both ways and checks
      that everything the JSON asserts is findable in the text.
- [x] Existing successful outputs remain unchanged; error-text changes are intentionally updated.
      The only deliberate wording changes are the runtime catalogue, which now names types in
      the language's vocabulary instead of Go's, and argument numbering, which counts from one.
- [x] Tests cover Unicode/CRLF positions, nested calls, failures after variable-length
      instructions, debug-on/off artifacts, and errors in nested function chunks.
      `TestFailureAfterVariableLengthInstructions` and `TestSourceMapIsKeyedAtInstructionStarts`
      cover the third: the latter walks the bytecode and rejects a location keyed inside an
      operand, which would look up successfully and report the wrong construct.
- [x] No new runtime import depends on a package missing from the embedded build bundle.
      `TestBuildBundleIsClosed` parses the bundled imports rather than waiting for a build to
      fail on someone's machine.

Phase 1 is complete. Two things it deliberately did not fix, for the phase that
owns them:

- The expected-token set in a syntax error is still recovered from Participle's
  message text, because the library keeps the set unexported. Everything else
  reads typed errors.
- Lenient and strict modes still disagree about what a mismatched conditional
  evaluates to. That is inference behavior, which Phase 4 settles.

## 6. Phase 2 — Language specification and semantic contracts

### Outcome

Create `docs/SPEC.md`, `docs/decisions/`, and `docs/MIGRATION.md`. Separate rules
already implemented from rules targeted by later phases. Each normative rule has
at least one executable conformance fixture. Phase 2 does not silently implement
all proposed breaking changes.

### P2.1 — Record the language grammar and value model

The specification is [docs/SPEC.md](SPEC.md). Every normative rule in it carries an
identifier, and `TestSpecRulesHaveFixtures` fails if one is added without a fixture.

- [x] Document lexical rules, comments, string escapes, identifiers, reserved words,
      boolean aliases, literals, and the role of whitespace. Section 1, `LEX-*`.
- [x] Specify operator precedence/associativity, calls, indexing, conditional expressions,
      function literals, lists, dictionaries, and blocks. Sections 2–3, `GRM-*`.
- [x] Define the difference between an expression value and statement output.
      `EVL-TOP-LEVEL-DISPLAY`, `EVL-PRINT`, `EVL-BLOCK-RESULT`.
- [x] Specify declaration visibility, shadowing, evaluation order, function arity,
      block results, closure capture, and recursion. Sections 6–7, `EVL-ORDER-*` and `BND-*`.
- [x] Specify indexing, missing-key failures, equality, printing, and integer boundaries.
      Section 5, `VAL-*`, and section 6.3, `EVL-DISPLAY-*`.
- [x] Define each configuration mode and which errors it can downgrade or disable.
      Section 9, `MOD-*`. `MOD-SEVERITY-ONLY` is asserted through warning fixtures,
      not only through outcomes, because a warned-about program still runs.
- [x] State which behaviors are intentionally unspecified; do not inherit Go behavior
      accidentally. Section 10, `UNS-*`. `TestUnspecifiedEntriesAreNotRules` keeps
      the list from acquiring fixtures, which would make it normative by accident.

### P2.2 — Record the proposed decisions

These defaults make the plan actionable. Implement a changed rule only alongside
its decision record, migration explanation, and affected tests.

Every row below now has a record in [docs/decisions/](decisions/), indexed in
[decisions/README.md](decisions/README.md), and every row that changes an existing
program has a before-and-after section in [docs/MIGRATION.md](MIGRATION.md).
`TestDecisionsAreLinked` checks that each record names rules the specification still
declares, that every planned rule has a record, and that every migration a record
promises is written. Writing the specification added two decisions the table does not
have: D-11 (unary negation) and D-17 (inference), for the reasons the index gives.

| Topic | Recommended target | Compatibility treatment / implementation phase |
| --- | --- | --- |
| Top-level expression output | Preserve automatic display of non-void results; `print(value)` has immediate effects and returns void | Preserve current successful scripts |
| Blocks | Permit declarations and expressions; last expression is the result, or void when the block ends with a declaration | New syntax in Phase 3 |
| Empty braces | Keep `{}` as an empty dictionary | Preserve current parsing; document how a block yields void |
| Statement boundaries | Keep newlines as whitespace; add an explicit optional `;` separator for otherwise ambiguous adjacent expressions | Phase 3; specify `f\n(x)` as a call, not automatic semicolon insertion |
| Evaluation order | Evaluate callee then arguments left to right; dictionary pairs in source order, key before value | Preserve and test side effects |
| Bindings | Immutable `let`; reject duplicate declarations in one scope; allow inner-scope shadowing | Phase 3; document departure from current map overwriting |
| Closure lookup | A resolved identifier keeps the same binding identity throughout execution | Phase 3; remove accidental late name rebinding |
| Forward references | Ordinary bindings must be declared before use; a function may refer to itself | Typed self-recursion in Phase 3; mutual forward recursion deferred |
| Integers | Signed 64-bit integers; checked arithmetic; explicit overflow errors | Phase 3; migrate AST/constants/runtime values from Go `int` |
| Division/remainder | Truncate quotient toward zero; remainder follows dividend sign; zero divisor is an error | Phase 3; `MinInt64 / -1` errors, `MinInt64 % -1` is zero |
| Booleans | `!`, `&&`, `\|\|` require bool and return bool; `&&`/`\|\|` short-circuit | New behavior in Phase 3; lenient `if` truthiness is a separate rule |
| Equality | Value equality for scalars and recursively comparable lists/dictionaries; reject functions and containers with function values | Specify now, implement without host reflection in Phase 3 |
| Collections | Homogeneous types; immutable operations; keys limited to int/string/bool; last duplicate key wins | Preserve core behavior; library operations in Phase 5 |
| Conversion | No implicit string/number coercion; explicit conversion functions | Preserve; add library functions in Phase 5 |
| Display | Specify stable scalar/container rendering and placeholder representations for void/functions; never expose host pointers | Phase 3 shared value semantics; preserve existing scalar example output |

The fixed-width, checked-integer policy is a Flux design choice. Go's signed
overflow behavior does not automatically provide that policy, so operator helpers
must check boundaries explicitly. [Host arithmetic reference](https://go.dev/ref/spec#Arithmetic_operators).

### P2.3 — Build the conformance harness

The fixture format is defined in `internal/conformance`; the harness is
`conformance_test.go`. A fixture is an ordinary Flux file whose leading `//!`
comments declare what it requires, so the file the parser reads is the file a
reader reads.

- [x] Add `testdata/conformance/` fixtures with source, required phase, configuration,
      expected output/value, error code, and source span. 100 fixtures across eight
      sections; a span is required for every declared failure.
- [x] Run each current-language fixture against interpreter and VM. Run representative
      fixtures through the real standalone compiler as well.
      `TestConformanceStandaloneExecutables` builds seven of them into real executables
      and checks the position each reports after the source is no longer consulted.
- [x] Store future examples separately with explicit milestone metadata; do not silently
      skip arbitrary failing tests or mark unsupported syntax as implemented. A planned
      fixture declares both what its rule requires and what Flux does today, and is run
      exactly like an implemented one; `TestPlannedRulesStillDifferFromTheSpecification`
      fails when the two converge, so a rule cannot become "done" without being promoted.
- [x] Give runtime-error fixtures a mode where they reach execution despite static checks.
      `TestConformanceCoversDiagnosticCodes` requires a fixture for every registered code,
      which is only satisfiable for the `R_` group by reaching execution under warn-only
      or disabled checking. Two codes are exempt with recorded reasons.
- [x] Add paired examples for ambiguous cases: shadowing, function comparisons, empty
      collections, duplicate keys, top-level output, and block result selection.
- [x] Maintain both expected-result assertions and backend parity assertions. Agreement
      between two implementations alone is not proof of correct semantics.
      `TestConformanceBackendParity` compares the engines to each other separately from
      the declared outcomes, and reports a disagreement as its own failure.

Lexical resolution is a semantic decision, not just an optimization. The resolver
must identify the intended declaration before evaluation so closure behavior cannot
change with unrelated environment mutations. [Binding reference](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/resolving-and-binding.md).

### Phase 2 completion criteria

- [x] Each row above has a decision record, examples, and an assigned implementation slice.
- [x] Existing conformance fixtures pass; future ones are clearly labeled as specifications.
- [x] No unresolved decision blocks arithmetic, block parsing, recursion, or type-mode work.
      D-09 to D-13 settle arithmetic and equality, D-02 blocks, D-08 recursion, and D-17
      the inference behavior Phase 4 depends on.
- [x] Breaking differences have concise before/after migration examples.
- [x] The README describes the implemented subset without promising future features.

## 7. Phase 3 — Core expressions, lexical scopes, and recursive functions

### Outcome

Ordinary arithmetic and logical expressions work consistently. Blocks support local
bindings, closures preserve lexical identity, and explicitly typed functions can
call themselves. Keep this separate from the inference overhaul in Phase 4.

### P3.1 — Add precedence levels and checked value operations

Target precedence, highest first:

| Level | Syntax | Associativity |
| --- | --- | --- |
| Postfix | calls, indexing | Left |
| Unary | `-`, `!` | Right |
| Multiplicative | `*`, `/`, `%` | Left |
| Additive | `+`, `-` | Left |
| Relational | `<`, `<=`, `>`, `>=` | One relational comparison per level; require parentheses for chaining |
| Equality | `==`, `!=` | Left; operand comparability still applies |
| Logical AND | `&&` | Left, short-circuit |
| Logical OR | `\|\|` | Left, short-circuit |

- [ ] Add longest-match lexer rules for multi-character operators before their prefixes.
- [ ] Keep minus as an operator token; do not make `1-2` lex as `1` and a signed literal.
- [ ] Replace the current single comparison level and extend AST grammar without
      introducing left recursion into Participle.
- [ ] Support parenthesized conditionals/functions wherever a primary expression is valid.
- [ ] Introduce runtime value helpers, proposed under `value/`, for arithmetic, comparison,
      equality, and stable printing. Do not centralize AST traversal or VM dispatch.
- [ ] Include shared runtime helpers in the Phase 1 embedded-source manifest.
- [ ] Parse integer magnitudes with explicit range handling. Support the literal
      `-9223372036854775808` without first rejecting its positive magnitude.
- [ ] Check addition, subtraction, multiplication, negation, division, remainder, and
      conversion boundaries in both normal evaluation and any future constant folding.
- [ ] Range-check list indexes before converting int64 to host `int`.
- [ ] Update serialized constants, gob tests, dictionaries, and existing integer assertions.
- [ ] Replace `reflect.DeepEqual` with the specified comparable-value semantics and
      explicit errors for unsupported equality operands.

### P3.2 — Implement short-circuit control flow

- [ ] Give logical expressions dedicated AST/checker/evaluator handling; do not send them
      through an eager generic binary-operator evaluator.
- [ ] Check both operand types statically, but evaluate the right operand only when required.
- [ ] Compile explicit branch paths that each produce one bool. With the current consuming
      jump opcode, emit a replacement `false`/`true` result on the skipped branch.
- [ ] Record source maps for generated branch instructions and any runtime bool checks.
- [ ] Verify jump targets and stack effects for nested expressions in arguments, arrays,
      dictionary values, and function returns.

The book's VM example uses jumps that retain the condition; Flux currently consumes
it. Copying its bytecode sequence unchanged would reintroduce a stack bug. Adapt
the control-flow technique to Flux's contract. [Short-circuit reference](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/jumping-back-and-forth.md).

### P3.3 — Resolve local bindings and capture environments

Proposed package: `resolver/`. Its output associates identifier uses with stable
binding IDs and records declarations, scopes, and captures. It also provides the
foundation for editor navigation in Phase 5.

- [ ] Allow block items to contain `let` statements and expressions while preserving
      dictionary/block disambiguation and the empty-dictionary rule.
- [ ] Add optional semicolons to grammar/token handling and retain their source positions.
- [ ] Resolve names before optional type checking, so disabling types does not change scope.
- [ ] Diagnose same-scope duplicates, invalid self-initialization, and unresolved names.
- [ ] Use declaration state to reject `let x = x` while allowing recursive function bodies.
- [ ] Bind locals separately from globals in both backends; locals must not escape their scope.
- [ ] Map identifiers to binding identities rather than repeatedly searching mutable name maps.
- [ ] Represent captured values through explicit environment cells or equivalent resolved
      slots; ensure their lifetime outlasts the outer function call.
- [ ] Add VM local/capture operations and function capture metadata as needed; keep
      four-byte operands and check bounds.
- [ ] Preserve one expression result at block exit while cleaning up its temporary values.

Captured cells are recommended for the initial implementation because Flux runs on
Go's managed heap and self-recursive closures need a stable binding to initialize.
Stack-slot/upvalue optimization can follow profiling. [Closure implementation reference](https://raw.githubusercontent.com/munificent/craftinginterpreters/master/book/closures.md).

### P3.4 — Introduce typed self-recursion

- [ ] Accept self-reference only for function-valued declarations, not arbitrary initializers.
- [ ] Require a complete signature for recursive functions in this phase, supplied by
      the variable annotation or parameter-plus-return annotations.
- [ ] Bind that signature before checking the body, then verify the body against it.
- [ ] Allocate the runtime binding before constructing the closure; initialize it before
      exposing the function to calls. Resolve self-reference to that exact binding.
- [ ] Handle both global and locally declared recursive functions.
- [ ] Add a configurable/internal execution depth limit with a structured failure so tests
      cannot exhaust the Go stack; define the user-facing default in the specification.
- [ ] Carry useful call traces through recursion; cap rendered trace length independently
      of the execution-depth limit.
- [ ] Defer mutual recursion and general forward-declaration hoisting to a later design.

Target example:

```flux
let factorial = fn(n: int): int => {
  let base: bool = n <= 1
  if base then { 1 } else { n * factorial(n - 1) }
}
print(factorial(5)) // 120
```

### Phase 3 completion criteria

- [ ] Precedence fixtures include `1 + 2 * 3`, `10 - 3 - 2`, `-2 * 3`, `!(1 < 2)`,
      and combinations of comparisons/equality/boolean operators.
- [ ] `false && (1 / 0 > 0)` and `true || (1 / 0 > 0)` succeed without evaluating division.
      Reversing their left booleans produces the expected runtime failure.
- [ ] Skipped branches produce no observable prints or calls; their static type errors
      are still reported when checking is enabled.
- [ ] Integer fixtures cover both extrema, overflow for each operator, signed remainders,
      zero divisors, and unary literal boundaries.
- [ ] Closure tests cover nested captures, shadowing, capture after outer return, and
      functions declared before a later inner binding of the same name.
- [ ] Typed factorial and a locally recursive function pass in all execution paths.
- [ ] Out-of-scope reads, duplicate declarations, bad recursion signatures, and excessive
      call depth report the designated diagnostic.
- [ ] Comparisons and printing of function values never expose Go implementation details.
- [ ] Existing examples, four-byte operand stress tests, and standalone builds still pass.

## 8. Phase 4 — Constraint-based inference and reliable checking modes

### Outcome

The checker records relationships between values instead of treating all unknowns
as compatible. Plain identity and collection functions can be reused safely;
ambiguous overloaded operations have explicit rules.

This is a deliberately limited inference system inspired by Hindley–Milner, not a
claim of full HM soundness/completeness for Flux's lenient/dynamic modes or overloads.

### P4.1 — Separate type identity, inference, recovery, and dynamic values

Proposed files: `types/variables.go`, `types/constraints.go`, `types/unify.go`,
`types/infer.go`, and later `types/schemes.go`.

- [ ] Separate structural type equality from assignability/compatibility policy.
- [ ] Introduce unique inference variables with bindings/substitutions.
- [ ] Add an error-recovery type that suppresses cascades but never makes a failed
      program count as successfully checked.
- [ ] Add a distinct dynamic type for deliberate lenient fallback; never use it as
      an inference variable or silently introduce it in strict mode.
- [ ] Keep existing public annotations (`int`, `string`, `bool`, `void`, list/dict/function)
      while adding internal type schemes. Explicit generic syntax is not required yet.
- [ ] Make inference state per analysis request so editor documents and separate runs
      cannot constrain one another.

### P4.2 — Solve constraints with diagnostic provenance

- [ ] Generate constraints for calls, returns, annotations, indexing, collections,
      conditions, and each operator's legal signatures.
- [ ] Store the originating expression and related declaration on each constraint.
- [ ] Implement substitutions and recursive unification of function/list/dictionary types.
- [ ] Reject infinite types with an occurs check; never silently bind a variable to a
      type containing that same variable.
- [ ] Define deterministic constraint processing and stable type-variable display names.
- [ ] Report the earliest useful conflict with related evidence, then recover locally.
- [ ] Keep a small, direct solver until profiling demonstrates a need for a more complex
      representation; benchmark adversarial nesting and conflicting constraints.

Constraint generation and unification supply the basic mechanism; this repository
must additionally preserve source origins and its n-argument function representation.
[Inference mechanism reference](https://people.cs.umass.edu/~arjun/courses/compsci631-fall2017/reading/lecture8.pdf).

### P4.3 — Define overloaded operations without accidental defaulting

Flux's `+` supports both integers and strings. Ordinary type equality alone cannot
choose between those operations for `fn(x) => x + x`.

- [ ] Model a finite set of admissible signatures for overloaded operations.
- [ ] Narrow that set using parameter/return annotations and other constraints.
- [ ] Do not generalize unresolved overloaded signatures into unrestricted type schemes.
- [ ] In strict mode, require an annotation when multiple overloads remain possible at
      the declaration boundary; never guess `int` or pick the first call's type.
- [ ] In lenient mode, preserve such a function as dynamic with an explicit warning and
      runtime operand checks; do not infer an incorrect concrete return type.
- [ ] Make unlike-branch lenient results dynamic, not an invented concrete branch type.
- [ ] Define comparable/key constraints for collection operations and reject functions
      where those operations cannot support them.

Unconstrained but valid polymorphism is different from unresolved overloading:
`fn(x) => x` can become a type scheme, while an ambiguous `+` still needs a policy.

### P4.4 — Add let-polymorphism and inferred self-recursion

- [ ] Introduce schemes that quantify suitable variables at immutable `let` bindings.
- [ ] Generalize only variables not free in the surrounding resolved type environment.
- [ ] Instantiate fresh variables at every scheme use; keep function parameters
      monomorphic within an individual function body.
- [ ] Test that captured outer parameters cannot be incorrectly generalized.
- [ ] Infer a self-recursive function through one provisional monomorphic signature,
      solve its body, then consider generalization after the recursive check succeeds.
- [ ] Reject polymorphic recursion; retain annotations as a clear way to constrain recursion.
- [ ] Infer empty collection element/key/value types from context or a valid generalized
      binding; never use void as an empty collection's inferred element type.
- [ ] Keep all user-observable bindings and collection operations immutable. Revisit
      generalization before introducing reference cells or mutation to the language.

Generalization and fresh instantiation let an identity binding work at distinct
call types without sharing one mutable inference variable. The surrounding
scope still constrains which variables can be generalized. [Cornell reference](https://cs3110.github.io/textbook/chapters/interp/inference.html).

Mutation would require additional restrictions; the OCaml manual demonstrates why
unrestricted generalization of stateful values can be unsound. [Mutability reference](https://ocaml.org/manual/5.3/polymorphism.html).

### P4.5 — Publish checking-mode guarantees

| Mode | Proposed guarantee |
| --- | --- |
| Strict | Solve required constraints; reject mismatches and unresolved overloads; allow valid generalized schemes; no implicit dynamic fallback |
| Lenient | Reject concrete unsafe arithmetic/annotation conflicts; allow documented truthiness, branch differences, and unresolved-overload fallback with warnings |
| Warn-only | Downgrade type diagnostics according to policy; execute using ordinary runtime checks; never silently insert conversions |
| Disabled | Skip type inference/checking; retain parsing, lexical name resolution, and runtime validation |

- [ ] Make policy independent of the solver's core type operations.
- [ ] Preserve diagnostics when downgraded: code, span, and cause must not disappear.
- [ ] Document that syntax/binding failures are structural and cannot be downgraded by
      warn-only. Earlier rejection of unresolved names is an intentional migration change.
- [ ] Ensure disabling checking does not change the values produced by valid programs.
- [ ] Explain that strict typing cannot guarantee successful execution: indexing,
      conversion, resource limits, and checked arithmetic can still fail at runtime.

### Phase 4 completion criteria

- [ ] `let id = fn(x) => x` can be used at int and string in the same program.
- [ ] `let dec = fn(x) => x - 1` infers an integer input/output; `dec("x")` is rejected.
- [ ] A typed or otherwise constrained string concatenation function remains valid.
- [ ] Ambiguous `fn(x) => x + x` follows the documented strict/lenient policy.
- [ ] `fn(x) => x(x)` fails with an infinite-type diagnostic instead of looping.
- [ ] Nested captured variables retain their constraints; scheme uses get fresh variables.
- [ ] Inferred self-recursion passes and inconsistent recursive calls fail.
- [ ] Empty/nested collections, higher-order functions, and wrong return annotations
      have positive and negative tests under all four modes.
- [ ] Inference failures retain source locations and do not contaminate the next analysis.
- [ ] Existing valid runtime behavior remains consistent across both backends and binaries.

## 9. Phase 5 — Standard library, checking, formatting, and editor tools

Ship this phase as four separately reviewable capabilities. The analysis interface
should be shared; each command or editor adapter should not reinvent parsing and
type checking.

### P5.1 — Shared analysis service and `flux check`

Proposed package: `analysis/`, accepting source snapshots and an explicit config
snapshot. Return syntax/resolution results, inferred types, and diagnostics.

- [ ] Extract parse → resolve → infer/check orchestration from `main.go`.
- [ ] Keep analysis free of evaluation, filesystem writes, and process execution.
- [ ] Add `flux check <file...>` with deterministic diagnostics across multiple independent
      files; multiple files do not imply imports or a shared global namespace.
- [ ] Add `--strict` and `--warnings-as-errors` as explicit overrides with documented precedence.
- [ ] Add `--format=json` returning one versioned JSON result object, including an empty
      diagnostic list on success. Machine output must not mix progress text with JSON.
- [ ] Preserve working-directory config discovery initially; support an explicit config
      path so editor requests need not change process working directory.
- [ ] Add stdin analysis with an explicit display filename for editor/tool integration.
- [ ] Document exit codes and the interaction between `--strict`, warn-only, and disabled
      configuration; explicit `--strict` enables checking and disables warn-only.
- [ ] Reject malformed/contradictory CLI options with a tool diagnostic.

Acceptance: checking a program containing `print("side effect")` produces no program
output, never executes user code, never creates `dist/`, and reports the same static
diagnostics as the execution commands under the same effective configuration.

### P5.2 — Small, specified standard library

Proposed package: `stdlib/`. Builtin metadata should define names, documentation,
arity, type schemes/overloads, and runtime entry points. Both backends must expose
the same catalogue and effects; backend-specific closure invocation can use adapters.

Start with these APIs; signatures below are specification notation, not new Flux
generic syntax:

| Builtin | Target behavior | Required edge cases |
| --- | --- | --- |
| `print(value)` | Preserve one argument and void return; use canonical value display | Side effects inside nested calls; containers and void |
| `len(value)` | Count list elements, dictionary entries, or Unicode code points in a string | Empty values; multibyte text; reject unsupported values |
| `str(value)` | Convert int/bool/string explicitly; no implicit container/function conversion | Negative integers; bool spelling; string identity |
| `parseInt(text)` | Parse a signed base-10 int64 or return a structured runtime error | Invalid digits, whitespace policy, sign-only input, overflow |
| `append(list, item)` | Return a new homogeneous list | Original list unchanged; empty-list inference |
| `keys(dict)` | Return a list of keys with deterministic order | Define numeric/string/bool ordering; empty dictionaries |
| `values(dict)` | Values in the corresponding `keys` order | Alignment with keys; homogeneous value typing |
| `hasKey(dict, key)` | Return bool without a missing-key error | Wrong key type; absent keys |
| `getOr(dict, key, fallback)` | Return found value or supplied fallback of the value type | Fallback is eagerly evaluated under ordinary call semantics |

- [ ] Give builtin values an explicit runtime representation rather than magic strings.
- [ ] Reuse Phase 4 schemes and constrained builtin overload handling; do not add a second
      inference algorithm inside the library.
- [ ] Inject output/context dependencies; builtins must not access hidden global state.
- [ ] Specify `parseInt` as decimal-only with no surrounding whitespace and optional sign;
      do not accidentally accept prefixes/underscores through host defaults.
- [ ] Distinguish code-point length from byte length and grapheme-cluster count in docs.
- [ ] Preserve immutability even when Go slices have spare capacity; test aliasing.
- [ ] Add the library and shared runtime dependencies to standalone bundling.
- [ ] Generate or validate builtin documentation against the same metadata used by the checker.

The host parser exposes explicit base/bit-size parameters and syntax/range errors;
wrap them with Flux's declared conversion policy and diagnostics.
[Go `strconv.ParseInt` documentation](https://pkg.go.dev/strconv#ParseInt).

Acceptance: every builtin has positive, type-error, runtime-error where relevant,
backend-parity, and standalone-executable tests. Optional `map`/`filter`/`fold` can
follow once higher-order builtin invocation has its own reviewed design.

### P5.3 — Comment-preserving formatter

Proposed package: `formatter/`, operating on syntax plus retained tokens/source.
Formatting must not require a successfully typed program.

- [ ] Define one formatting style: indentation, line width, line breaks, spacing,
      collection layout, and final-newline policy.
- [ ] Attach comments as leading, trailing, or internal trivia with stable positions.
- [ ] Preserve comment content and order; do not lose comments between operators or
      around otherwise empty collections.
- [ ] Print parentheses according to precedence and associativity; preserve grouping
      where removing it could change parsing.
- [ ] Emit explicit separators where adjacent expression statements would otherwise
      merge into a call, index, or binary expression.
- [ ] Add `flux fmt <file>` to print formatted source and `--write` for explicit edits.
- [ ] Add `flux fmt --check <file...>` for CI; define success/difference/tool-error exits.
- [ ] Accept explicit files and directories; expand directories to `.flux` files in a
      deterministic order, excluding `.git/` and `dist/` and not following symlinks.
- [ ] On invalid syntax, report diagnostics and leave files unchanged. Type errors alone
      do not prevent formatting syntactically valid code.
- [ ] Validate all requested files before write mode begins; document per-file atomic
      replacement rather than promising a cross-file transactional write.
- [ ] Keep source maps/diagnostics based on the new parsed snapshot after an edit.

Required properties:

```text
format(format(source)) == format(source)
normalize(parse(format(source))) == normalize(parse(source))
run(format(source)) == run(source)  // bounded valid conformance programs
```

Normalization removes positions/trivia and equivalent parenthesis wrappers, not
meaningful syntax. Also check comment preservation separately: AST equivalence
alone cannot detect lost comments. Comments and semantic checking are recognized
formatter concerns. [Comment rationale](https://prettier.io/docs/rationale.html#comments),
[formatter correctness checks](https://prettier.io/docs/cli.html#--debug-check).

Acceptance: idempotence, normalized-tree equality, comment retention, invalid-file
preservation, and representative interpreter/VM behavior all pass, including
strings with escapes, CRLF, nested blocks/dictionaries, and statement boundaries.

### P5.4 — Go language server and VS Code client

Proposed command: `flux lsp`; proposed package: `lsp/`. Add a small TypeScript client
under `vsce/src/` and keep the current grammar available when no server is running.

Start with a documented LSP 3.17 subset:

- [ ] Implement initialize/initialized, shutdown/exit, and stdio JSON-RPC framing.
      Stdout is reserved for protocol messages; diagnostics/logs go through the protocol
      or stderr, never ad hoc terminal output.
- [ ] Negotiate position encoding; support UTF-16 and optionally UTF-8 using Phase 1's
      source-index conversions. Do not reuse rune columns as UTF-16 offsets.
- [ ] Support didOpen/didChange/didClose using in-memory, versioned document snapshots.
      Start with advertised full synchronization; incremental sync is an optimization.
- [ ] Analyze unsaved text using the shared analysis service and explicit config snapshots.
- [ ] Publish diagnostics with severity, code, source, ranges, and related declaration locations.
- [ ] Clear previous diagnostics with an empty result when corrected; clear single-file
      diagnostics when the server receives document-close ownership notification.
- [ ] Debounce updates and discard results for obsolete document versions; add cancellation
      and per-request analysis state before concurrent processing.
- [ ] Start with diagnostics even if parsing fails. Add bounded syntax recovery for local
      completion later; never feed recovered syntax to execution.
- [ ] Add hover from inferred types, completion from visible bindings/builtins, and
      definition navigation from the resolver's binding identities.
- [ ] Delegate document formatting to the same formatter as the CLI.
- [ ] Resolve config per workspace/document without process-global `chdir`; define an
      explicit default for untitled documents and isolated multi-root workspaces.
- [ ] Add a configurable server executable path and a clear missing/incompatible-server
      message. Do not implicitly download executable dependencies as part of analysis.
- [ ] Pin a compatible `vscode-languageclient` release, add lockfiles/build scripts, and
      validate the declared minimum VS Code version.
- [ ] Add protocol-transcript tests and extension-host tests for open/edit/fix/close,
      hover/definition, completion scope, formatting, and server restart.

The protocol specifies zero-based positions with negotiated encoding, diagnostic
objects with ranges/codes, and versioned document updates. Keep these as adapters
around Flux's internal analysis data. [Positions](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/types/position.md),
[diagnostic shape](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/types/diagnostic.md),
[document updates](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/textDocument/didChange.md).
The server must replace/clear published diagnostics rather than leave stale results.
[Publishing contract](https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.17/language/publishDiagnostics.md).

The client/server split keeps language implementation in Go and editor integration
in a thin extension. [VS Code implementation guide](https://code.visualstudio.com/api/language-extensions/language-server-extension-guide).

### Phase 5 completion criteria

- [ ] Static checking, execution, and editor analysis share one front-end service.
- [ ] Every proposed core builtin is documented and tested in all execution paths.
- [ ] Formatting is stable and preserves syntax, comments, and tested behavior.
- [ ] Editor diagnostics use unsaved content, handle Unicode correctly, and do not revert
      to stale errors after rapid edits.
- [ ] Editor features never evaluate user code or mutate source except an explicit formatting edit.
- [ ] CLI and extension packaging include the required runtime/server assets.
- [ ] Native integration checks cover supported operating systems where runners exist;
      cross-compilation alone is not recorded as successful native execution.

## 10. Verification strategy across phases

### Test matrix

| Layer | Essential coverage |
| --- | --- |
| Lexer/parser | Token boundaries, precedence, malformed syntax, escaped strings, comments, source positions, block/dictionary ambiguity |
| Resolver | Scope entry/exit, shadowing, duplicates, capture identity, self-initialization, self-recursion |
| Type system | All modes, annotations, inference variables, overload ambiguity, schemes, occurs check, recursive signatures |
| Interpreter/VM | Value/output/error parity, evaluation order, stack cleanup, bounds, checked integers, call-depth failures |
| Standalone artifact | Fresh-directory build/run, no source checkout, offline module resolution, nested function metadata, debug-on/off diagnostics |
| Formatter | Idempotence, normalized syntax, comments, separators, atomic per-file writes, invalid-input preservation |
| LSP/client | Framing, lifecycle, unsaved buffers, versions, cancellation, encoding conversion, stale-diagnostic clearing, workspace isolation |

### Fuzzing and differential execution

- Add parser fuzzing first: arbitrary bytes must produce a bounded result or diagnostic,
  never an unexpected panic or uncontrolled recursion.
- Use bounded generated valid programs for interpreter/VM differential testing. Fix
  maximum AST depth, collection size, recursion depth, and execution work per case.
- Enforce deterministic fuel/depth limits inside execution; a goroutine timeout alone
  cannot safely stop an evaluator that continues running.
- Compare semantic values and stable error codes/locations as well as stdout.
- Exclude wall-clock timing and unstable paths from equality assertions.
- Persist minimized failures as ordinary fixtures or Go fuzz corpus entries.
- Add formatter round-trip and source-position conversion fuzz targets in their phases.
- Keep fuzz state local to the invocation. [Go fuzzing guidance](https://go.dev/doc/security/fuzz/).

### Commands and CI gates

Existing checks remain applicable:

```sh
go test ./...
go test -race -coverpkg=./... ./...
go vet ./...
gofmt -s -l .
make build-all
```

Formatting output must be empty; merely running `gofmt -l` is not a pass/fail gate.
Retain a shell assertion in CI. Run the minimum supported Go version and the
selected current development toolchain; update that matrix deliberately.

Proposed targets, to add when their tests exist:

```sh
go test ./parser -run='^$' -fuzz=FuzzParse -fuzztime=30s
go test ./formatter -run='^$' -fuzz=FuzzFormat -fuzztime=30s
go test . -run TestConformance -count=1
go test ./lsp -run TestProtocol -count=1
flux fmt --check examples
```

Use short deterministic/seed tests on every PR and longer fuzz/performance runs on
scheduled jobs. Coverage is evidence of exercised code, not a substitute for the
semantic matrix. Track branch/error coverage in the solver and VM without choosing
an arbitrary percentage as the sole release criterion.

Measure parser/checker allocations and latency on fixed fixtures at each phase.
For editor work, establish an initial diagnostic-latency budget on a recorded
machine and corpus, then gate measured regressions. Do not promise a universal
latency number before profiling the current grammar and solver.

## 11. Suggested implementation change sets

Each row is a reviewable slice; a large row can be split while preserving its gate.

| Order | Change set | Gate |
| --- | --- | --- |
| 1 | Source snapshots/spans and parser positions | Unicode/EOF position fixtures |
| 2 | Structured checker diagnostics and renderers | Existing checker tests migrated |
| 3 | Runtime source bundle and serialization version | Offline fresh-directory executable test |
| 4 | Returned runtime errors, VM source maps, traces, output injection | Three execution paths agree on failures |
| 5 | Specification, decisions, migration notes, conformance manifest | Current behavior documented; target changes explicit |
| 6 | int64 value semantics, unary/multiplicative/comparison operators | Numeric boundary and precedence tests |
| 7 | Short-circuit booleans | Skipped-side effects/errors; VM stack invariants |
| 8 | Block statements, separators, resolver, local/captured bindings | Scope and capture conformance |
| 9 | Typed self-recursion and bounded call depth | Global/local recursion and depth errors |
| 10 | Constraint variables, unification, overload policy | Solver and four-mode matrix |
| 11 | Type schemes and inferred self-recursion | Fresh instantiation and inference examples |
| 12 | Shared analysis service and `flux check` | No evaluation; stable JSON contract |
| 13 | Builtin catalogue and standard library | Backend and standalone library parity |
| 14 | Formatter and CLI | Round-trip, comments, idempotence |
| 15 | LSP diagnostics and VS Code client | Unsaved Unicode edits and protocol tests |
| 16 | Hover, completion, definitions, editor formatting | Resolver/type-backed extension-host tests |

## 12. Risks, decisions to preserve, and release checklist

| Risk | Mitigation |
| --- | --- |
| Two backends drift as syntax grows | One conformance corpus plus expected results; share value operations, not execution control flow |
| Shared packages break generated executables | Source-bundle manifest and offline artifact test before each dependency change |
| More grammar ambiguity causes poor parse performance | Keep ambiguity fixtures and benchmarks; profile before replacing the parser |
| int64 migration misses an assertion or gob type | Explicit numeric migration checklist and serialized nested-function fixtures |
| Recursive closures capture uninitialized or wrong bindings | Resolved identities, declaration states, initialized cells, local recursion tests |
| Overloading is mistaken for simple polymorphism | Finite overload constraints; strict ambiguity errors; explicit dynamic fallback |
| Generalization loses enclosing constraints | Captured-variable and fresh-instantiation tests; no mutation in scope |
| Comment loss goes unnoticed by AST tests | Independent comment/trivia assertions and formatter goldens |
| LSP errors move after emoji or CRLF | One tested source-index adapter with encoding-specific fixtures |
| Unbounded inputs/recursion freeze tools | Parser/analysis depth budgets, deterministic execution fuel, request cancellation |
| New builtins collide with user names | Document ordinary lexical shadowing and migration notes before adding names |

Before releasing a phase:

- [ ] All phase-specific and cross-cutting completion criteria pass.
- [ ] No unrelated existing work has been overwritten or omitted from review.
- [ ] Specification sections clearly distinguish implemented and future behavior.
- [ ] Diagnostics/error codes and JSON format changes are versioned and documented.
- [ ] Breaking semantics have migration examples and an intentional release version.
- [ ] Executable and editor packages are validated at their applicable native/platform gates.
- [ ] Runtime/source-map/bytecode changes preserve or explicitly reject incompatible formats.
- [ ] Known limitations are listed without presenting cross-builds as runtime tests.
- [ ] Publishing is performed as a separate release action after local validation.

The first implementation slice is P1.1–P1.2: canonical source spans, named parsing,
and position fixtures. It establishes the data every later phase depends on.
