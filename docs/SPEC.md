# The Flux language specification

Status: normative for the implementation in this repository. Last revised
2026-09-18, alongside Phase 2 of [the development plan](PLAN.md).

This document says what a Flux program means. It is written to be checked
rather than believed: every normative rule below carries an identifier, and
every identifier has at least one fixture under
[`testdata/conformance/`](../testdata/conformance) that runs on both execution
engines. `TestSpecRulesHaveFixtures` fails if a rule is added here without one.

## How to read this document

Each rule is a list item that begins with its identifier and its status:

- `(implemented)` — Flux behaves this way today. Its fixture asserts the rule.
- `(phase N)` — Flux is specified to behave this way after phase N of the plan.
  Its fixture asserts what Flux does *now*, records what the rule requires, and
  fails once the two agree, so the rule cannot be marked done by accident.

A planned rule is not a promise about a release date. It is a decision that has
been made and written down, so that the implementation, the migration notes and
the tests cannot drift apart while it is being carried out. Every planned rule
has a record in [`docs/decisions/`](decisions/) explaining why, and every rule
that changes existing behavior has a before-and-after example in
[`docs/MIGRATION.md`](MIGRATION.md).

[Intentionally unspecified](#10-intentionally-unspecified) lists behavior that is
deliberately *not* a rule. Those entries are numbered `UNS-…`, have no fixtures,
and exist so that host behavior visible through Flux is never mistaken for a
guarantee.

Terminology: an **expression** produces a value. A **statement** is a top-level
declaration or expression. **void** is the type of an expression that produces
no value; `print` returns it. A **mode** is one of the four type-checking
configurations in [section 9](#9-type-checking-modes).

## 1. Lexical structure

A source file is UTF-8 text. It is read as a sequence of tokens; whitespace and
comments separate tokens and carry no other meaning.

- `LEX-WHITESPACE` (implemented) — Spaces, tabs, carriage returns and newlines
  separate tokens. A newline is ordinary whitespace: it neither ends a statement
  nor inserts one. Layout never changes meaning.
- `LEX-COMMENT-LINE` (implemented) — `//` begins a comment that runs to the end
  of the line.
- `LEX-COMMENT-BLOCK` (implemented) — `/*` begins a comment that ends at the
  first following `*/`. Block comments do not nest: in `/* a /* b */ c`, the
  comment ends before `c`.
- `LEX-IDENT` (implemented) — An identifier starts with an ASCII letter or `_`
  and continues with ASCII letters, digits and `_`. Identifiers are
  case-sensitive.
- `LEX-RESERVED` (implemented) — `if`, `then`, `else`, `let`, `fn`, `int`,
  `string`, `bool`, `void`, `true`, `false`, `yes` and `no` are reserved and
  cannot be used as identifiers. They are reserved in lower case only; `IF` is
  an ordinary identifier.
- `LEX-INT` (implemented) — An integer literal is one or more decimal digits.
  There is no sign, no base prefix and no digit separator: `-1` is not a
  literal (see `VAL-NEG`), and `1_000` is two tokens.
- `LEX-INT-RANGE` (implemented) — An integer literal that does not fit a signed
  64-bit integer is rejected while the file is read.
- `LEX-STRING` (implemented) — A string literal is delimited by `"`. It may
  span lines. `\` begins an escape sequence; the accepted set is
  [unspecified](#10-intentionally-unspecified) beyond `\\`, `\"`, `\n` and `\t`.
- `LEX-BOOL` (implemented) — `true` and `false` are boolean literals. `yes` and
  `no` are accepted as aliases for them and mean exactly the same thing.

## 2. Program and statement structure

- `GRM-PROGRAM` (implemented) — A program is zero or more statements, evaluated
  in order. An empty program is valid and produces no output.
- `GRM-STATEMENT` (implemented) — A statement is either a `let` declaration or
  an expression. Statements are not separated by punctuation; the grammar is
  unambiguous without it.
- `GRM-STATEMENT-SEPARATOR` (phase 3) — An optional `;` may separate adjacent
  statements, for the cases where juxtaposition reads as one expression.
- `GRM-ADJACENCY` (implemented) — Because a newline is whitespace, a `(`
  beginning the next line continues the expression before it: `f` followed by
  `(x)` on the next line is a call. Flux does not insert statement terminators.

## 3. Expressions

### 3.1 Precedence and grouping

Operators, loosest first:

| Level | Operators | Associativity |
| --- | --- | --- |
| 1 | `==`, `<`, `>` | left |
| 2 | `+`, `-` | left |
| 3 | postfix `(…)` call, `[…]` index | left |

- `GRM-PRECEDENCE` (implemented) — Comparison binds more loosely than addition
  and subtraction, so `a - b < c` is `(a - b) < c`.
- `GRM-ASSOCIATIVITY` (implemented) — Both levels associate to the left:
  `3 - 2 - 1` is `(3 - 2) - 1`, which is `0`.
- `GRM-GROUP` (implemented) — `( … )` groups an expression and overrides
  precedence. It produces the value of the expression inside it.
- `GRM-POSTFIX` (implemented) — Calls and indexes apply left to right to the
  expression before them, so `f(x)[0](y)` calls `f`, indexes the result, and
  calls that.

### 3.2 Conditionals, functions and calls

- `GRM-IF` (implemented) — `if C then A else B` is an expression. `else` is
  required; there is no one-armed `if`, because every expression has a value.
- `GRM-FN` (implemented) — `fn(p1, p2) => body` is a function literal.
  A parameter may carry `: T`, and the literal may carry a return annotation
  before `=>`: `fn(x: int): int => x + 1`.
- `GRM-CALL` (implemented) — `f(a, b)` calls `f` with the arguments in
  parentheses. A trailing comma is not permitted.
- `GRM-INDEX` (implemented) — `c[k]` reads element `k` of a list or dictionary.

### 3.3 Collections and blocks

- `GRM-LIST` (implemented) — `[a, b, c]` is a list literal. `[]` is the empty
  list. A trailing comma is not permitted.
- `GRM-DICT` (implemented) — `{k: v, k2: v2}` is a dictionary literal. A
  trailing comma is not permitted.
- `GRM-EMPTY-BRACES` (implemented) — `{}` is the empty dictionary, not an empty
  block. A block that yields void is written `{ print("") }` or any block whose
  last expression is void.
- `GRM-BLOCK` (implemented) — `{ e1 e2 e3 }` is a block: a sequence of
  expressions in braces. It is distinguished from a dictionary literal by its
  contents, and only the last expression's value survives.
- `GRM-BLOCK-DECLARATION` (phase 3) — A block may contain `let` declarations.
  A declaration is scoped to the block, and a block whose last item is a
  declaration produces void.

## 4. Types and annotations

Flux has four scalar types — `int`, `string`, `bool`, `void` — and three
composite forms: `[T]`, `{K: V}` and `fn(T, …) -> R`.

- `GRM-TYPE` (implemented) — A type is `int`, `string`, `bool`, `void`, a list
  type `[T]`, a dictionary type `{K: V}`, or a function type
  `fn(T, …) -> R`. Type syntax appears only in annotations.
- `TYP-ANNOTATION-LET` (implemented) — `let name: T = value` requires `value`
  to have type `T`, and binds `name` at type `T` regardless of what was
  inferred.
- `TYP-ANNOTATION-PARAM` (implemented) — An annotated parameter has the
  annotated type inside the body, and calls must supply a matching argument.
- `TYP-ANNOTATION-RETURN` (implemented) — An annotated return type must match
  the type of the body.
- `TYP-INFERRED` (implemented) — An unannotated `let` takes the type of its
  value. An unannotated parameter has no known type; the checker neither
  constrains it nor reports against it.
- `TYP-NO-COERCION` (implemented) — There is no implicit conversion between
  types. `1 + "1"` is an error in both directions, and no operator converts its
  operands.
- `TYP-INFERENCE-CONSTRAINTS` (phase 4) — An unannotated parameter is given a
  type variable, and its uses in the body constrain it, so `fn(x) => x + 1`
  is known to take and return `int`.

## 5. Values and operations

### 5.1 Scalars

- `VAL-INT` (implemented) — `int` is a signed 64-bit integer.
- `VAL-INT-OVERFLOW` (phase 3) — An operation whose result does not fit an
  `int` is an error. Today the result wraps silently.
- `VAL-STRING` (implemented) — `string` is an immutable sequence of bytes,
  written as a literal. Strings are not indexable.
- `VAL-BOOL` (implemented) — `bool` is `true` or `false`.
- `VAL-VOID` (implemented) — `void` is the type of an expression with no value.
  `print` returns void. Void cannot be operated on.

### 5.2 Arithmetic and comparison

- `VAL-ADD-INT` (implemented) — `+` on two `int`s adds them.
- `VAL-ADD-STRING` (implemented) — `+` on two `string`s concatenates them. It
  is the only overload of `+`; mixing an `int` and a `string` is an error.
- `VAL-SUB` (implemented) — `-` subtracts two `int`s. It has no other operand
  types.
- `VAL-ORDER` (implemented) — `<` and `>` compare two `int`s and produce a
  `bool`. They have no other operand types.
- `VAL-EQUALITY` (implemented) — `==` compares two values and produces a
  `bool`. Scalars compare by value; lists and dictionaries compare element by
  element. Values of different types are never equal.
- `VAL-EQUALITY-FUNCTION` (phase 3) — Comparing functions, or containers that
  hold functions, is an error rather than a structural comparison.
- `VAL-DIVIDE` (phase 3) — `/` divides two `int`s, truncating the quotient
  toward zero, and `%` takes the remainder, which has the sign of the dividend.
  A zero divisor is an error.
- `VAL-NEG` (phase 3) — Unary `-` negates an `int`.
- `VAL-LOGICAL` (phase 3) — `!` takes a `bool` and produces a `bool`; `&&` and
  `||` take two `bool`s, produce a `bool`, and do not evaluate their right
  operand when the left one decides the result.

### 5.3 Collections

- `VAL-LIST-HOMOGENEOUS` (implemented) — Every element of a list has the same
  type. `[]` has an as-yet-unknown element type.
- `VAL-LIST-INDEX` (implemented) — A list is indexed by an `int`. An index
  below zero or at or above the length is an error.
- `VAL-DICT-HOMOGENEOUS` (implemented) — Every key of a dictionary has the same
  type, and every value has the same type.
- `VAL-DICT-KEY-TYPE` (implemented) — A dictionary key is an `int`, a `string`
  or a `bool`. Lists, dictionaries and functions are not keys.
- `VAL-DICT-DUPLICATE` (implemented) — When a literal writes the same key
  twice, the last pair wins. Both values are still evaluated.
- `VAL-DICT-MISSING` (implemented) — Reading a key a dictionary does not hold
  is an error, not a default value.
- `VAL-IMMUTABLE` (implemented) — There is no operation that modifies a list or
  dictionary in place. A collection's contents are fixed once it is built.

### 5.4 Functions

- `VAL-FN-ARITY` (implemented) — A call must supply exactly as many arguments
  as the function declares.
- `VAL-FN-NOT-CALLABLE` (implemented) — Calling a value that is not a function
  is an error.

## 6. Evaluation

### 6.1 Order

- `EVL-ORDER-STATEMENT` (implemented) — Statements run in source order.
- `EVL-ORDER-CALL` (implemented) — A call evaluates the callee first, then its
  arguments left to right, then performs the call.
- `EVL-ORDER-LIST` (implemented) — List elements are evaluated left to right.
- `EVL-ORDER-DICT` (implemented) — Dictionary pairs are evaluated in source
  order, and within a pair the key is evaluated before the value.
- `EVL-IF-BRANCH` (implemented) — A conditional evaluates its condition, then
  exactly one branch. The other branch is never evaluated.

### 6.2 Results and output

- `EVL-PRINT` (implemented) — `print(v)` writes `v` followed by a newline to
  standard output, immediately, and produces void.
- `EVL-TOP-LEVEL-DISPLAY` (implemented) — A top-level expression statement
  whose value is not void displays that value the way `print` would. A `let`
  declaration displays nothing.
- `EVL-BLOCK-RESULT` (implemented) — A block produces the value of its last
  expression. Earlier expressions are evaluated for their effects, and their
  values are discarded.
- `EVL-DIAGNOSTIC-STREAM` (implemented) [TestCLI] — A program's output goes to standard
  output and every diagnostic goes to standard error, so one can be read
  without the other.
- `EVL-EXIT-STATUS` (implemented) [TestCLI] — A command exits `0` when it did what was
  asked, `1` when a Flux program was rejected or failed while running, and `2`
  when Flux could not run the command at all.

### 6.3 Display

- `EVL-DISPLAY-SCALAR` (implemented) — An `int` displays as its decimal digits.
  A `string` displays as its characters, without quotes or escapes. A `bool`
  displays as `true` or `false`, never as `yes` or `no`.
- `EVL-DISPLAY-CONTAINER` (phase 3) — A list displays as `[a, b, c]` and a
  dictionary as `{k: v}`, with nested strings quoted so that a displayed value
  can be told apart from a displayed name. Today both use a host-defined form;
  see `UNS-DISPLAY-CONTAINER`.
- `EVL-DISPLAY-OPAQUE` (phase 3) — A function or a void value displays as a
  fixed placeholder that contains no host address and is identical on both
  engines. Today neither is true; see `UNS-DISPLAY-OPAQUE`.

### 6.4 Conditions

- `EVL-IF-TRUTHY` (implemented) — Outside strict mode a condition that is not a
  `bool` is accepted with a warning and treated as truthy: `0`, `""` and void
  are false, and every other value is true.
- `EVL-IF-BOOL` (phase 3) — A condition must be a `bool` in every mode.
  Truthiness is removed.

## 7. Bindings, scope and lifetime

- `BND-LET` (implemented) — `let name = value` evaluates `value` and binds it
  to `name`. The binding is immutable: nothing assigns to a name after it is
  bound.
- `BND-TOP-LEVEL-SCOPE` (implemented) — A top-level binding is visible to every
  statement, and inside every function, from the point the declaration is
  evaluated onward.
- `BND-PARAMETER-SCOPE` (implemented) — A parameter is visible only inside its
  function's body, and shadows a top-level binding of the same name.
- `BND-DUPLICATE-PARAMETER` (implemented) — A function that declares the same
  parameter name twice is rejected.
- `BND-CAPTURE` (implemented) — A function literal captures the bindings in
  scope where it is written, so a function returned from another function keeps
  that function's parameters.
- `BND-BUILTIN-PRINT` (implemented) — `print` is predeclared with type
  `fn(T) -> void`. It is an ordinary binding: a program may shadow it.
- `BND-REDECLARE` (phase 3) — Declaring a name twice in one scope is rejected.
  Today the second declaration silently replaces the first.
- `BND-FORWARD-REFERENCE` (phase 3) — A name must be declared before it is
  used. Today the checker rejects a forward reference, but a program that
  reaches execution with checking relaxed resolves it against whatever the name
  holds by then.
- `BND-CAPTURE-IDENTITY` (phase 3) — A captured name keeps the binding it
  resolved to when the function was written, so redeclaring that name later
  cannot change what an existing function sees.
- `BND-SELF-RECURSION` (phase 3) — A function may call itself by the name it is
  being bound to, provided the binding carries a return-type annotation. Today
  the checker reports the name as undefined.

## 8. Diagnostics

- `DIA-CODE` (implemented) — Every diagnostic carries a code whose prefix names
  the stage that reported it: `S_` syntax, `B_` binding, `T_` type, `R_`
  runtime, `X_` a defect in Flux itself. Codes are stable; message wording is
  not. The catalogue is [docs/DIAGNOSTICS.md](DIAGNOSTICS.md).
- `DIA-LOCATION` (implemented) — Every diagnostic about a program underlines
  the construct that is wrong, not the construct after it.
- `DIA-ENGINE-PARITY` (implemented) [TestConformanceBackendParity] — A program that fails does so with the
  same code and the same location whether it was interpreted or compiled.

## 9. Type checking modes

A mode is selected by `flux.json`. See [README](../README.md#type-checking-modes)
for the file format.

| Mode | `enabled` | `strict` | `warnOnly` |
| --- | --- | --- | --- |
| strict | true | true | false |
| lenient (default) | true | false | false |
| warn-only | true | — | true |
| disabled | false | — | — |

- `MOD-DISABLED` (implemented) — Checking is skipped entirely. The program runs
  and reports whatever it encounters at run time.
- `MOD-LENIENT` (implemented) — Mismatches are errors, except for the three
  rules listed under `MOD-STRICT`, which are warnings.
- `MOD-STRICT` (implemented) — A non-`bool` condition (`EVL-IF-TRUTHY`),
  conditional branches of differing types, and `==` between different types are
  errors instead of warnings. No other rule changes.
- `MOD-WARN-ONLY` (implemented) — Every diagnostic the checker would report as
  an error is reported as a warning instead, and the program runs anyway. It
  usually then fails at run time, because the mismatch was real.
- `MOD-SEVERITY-ONLY` (implemented) — A mode changes only severity. The code,
  the message and the span of a diagnostic are identical in every mode, so a
  test may assert on the code and let the mode decide only whether the command
  fails.
- `MOD-OUTPUT-INVARIANT` (implemented) — A mode never changes what a program
  prints. It decides whether the program runs, not what it does.

## 10. Intentionally unspecified

These are not rules. They describe places where Flux currently exposes behavior
that comes from its host implementation and that a program must not rely on.
Each names the rule that will replace it.

- `UNS-STRING-ESCAPE` — Beyond `\\`, `\"`, `\n` and `\t`, the escape sequences
  a string literal accepts are those of the host's string unquoting, including
  `\xNN`, `\uNNNN` and octal forms. Which escapes exist, and what an
  unrecognized one does, is not yet decided.
- `UNS-DISPLAY-CONTAINER` — A displayed list or dictionary currently uses the
  host's default formatting (`[1 2 3]`, `map[a:1]`). `EVL-DISPLAY-CONTAINER`
  replaces it.
- `UNS-DISPLAY-OPAQUE` — A displayed function currently reveals a host address,
  and the two engines disagree on the text. A displayed void value currently
  appears as a host placeholder. `EVL-DISPLAY-OPAQUE` replaces both.
- `UNS-EQUALITY-FUNCTION` — `==` on functions currently compares host
  structures, so a function equals itself and differs from an identical
  literal. `VAL-EQUALITY-FUNCTION` replaces it.
- `UNS-INT-WIDTH` — `int` is specified as 64-bit (`VAL-INT`), but the
  implementation currently uses the host's `int`, so the width follows the
  build target. `VAL-INT-OVERFLOW` fixes both the width and the overflow
  behavior.
- `UNS-DIAGNOSTIC-TEXT` — The wording of a diagnostic message, the set of
  attached notes, and the phrasing of the grammar's "expected …" list are not
  stable. Assert on codes and spans instead.
- `UNS-DIAGNOSTIC-COUNT` — How many diagnostics the checker reports for one
  program is not specified; it recovers and continues, and the number of
  follow-on reports may change. The first diagnostic in source order is the
  one fixtures assert.
- `UNS-BYTECODE` — The compiled bytecode format has no compatibility promise.
  Rebuild after a compiler change; already-built executables are unaffected.
- `UNS-EVALUATION-DEPTH` — There is no specified recursion limit. A deeply
  recursive program may exhaust the host stack.
