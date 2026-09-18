# Flux Language

Flux is a small programming language implemented in Go. A program can be run
directly or compiled to a standalone executable; both paths resolve lexical bindings and use the same optional
type checker and report failures with the same codes and positions.

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/pranavms13/flux-lang)
## Author

**Pranav M S**
- GitHub: [@pranavms13](https://github.com/pranavms13)
- Email: [flux@pranavms.dev](mailto:flux@pranavms.dev)

## Language specification

[docs/SPEC.md](docs/SPEC.md) says what a Flux program means. Every rule in it has
an identifier and at least one fixture under
[`testdata/conformance/`](testdata/conformance) that runs on both the interpreter
and the VM, so the document and the implementation cannot drift apart.

A rule marked `(phase N)` is a decision that has been made and not yet carried
out; its fixture asserts what Flux does *today* and fails if that ever starts to
match the rule without the rule being marked done. The reasoning behind each
decision is in [docs/decisions/](docs/decisions/), and anything that will change
an existing program has a before-and-after example in
[docs/MIGRATION.md](docs/MIGRATION.md).

## What the language has today

This is the whole of it. Anything not listed is not implemented.

- **Values**: `int` (signed 64-bit), `string`, `bool` (`true`/`false`, with
  `yes`/`no` as aliases), `void`, lists, dictionaries and functions.
- **Operators**: checked `+`, `-`, `*`, `/`, `%` and unary `-` on signed
  64-bit integers; string concatenation with `+`; `==`, `!=`, `<`, `<=`, `>`,
  `>=`; boolean `!`, short-circuit `&&` and `||`. Parentheses group.
- **Expressions**: boolean `if C then A else B`, function literals
  `fn(x: int): int => body`, calls, list/dictionary indexing and literals,
  and blocks with local declarations and a final result.
- **Bindings and functions**: immutable `let` at top level or in blocks,
  optional semicolons, lexical shadowing, closures that retain captured
  bindings, and global/local self-recursion with complete type signatures.
  Non-void top-level expressions display their value.
- **Built-ins**: `print`, which writes one line and returns void. It is an
  ordinary binding and can be shadowed.
- **Type checking**: static, with four modes selected by `flux.json`, and
  annotations for every type form. An unannotated parameter is not constrained unless a variable signature supplies its type.
- **Diagnostics**: every failure carries a code and underlines the construct
  that is wrong, identically from both execution engines.

There are no loops, assignment, explicit `return`, mutual recursion, modules
or standard library beyond
`print`. [docs/SPEC.md](docs/SPEC.md) records which of those are decided for a
later phase and which are simply absent.

```flux
let factorial = fn(n: int): int => {
  let base = n <= 1
  if base then 1 else n * factorial(n - 1)
}
print(factorial(5)) // 120
```

See [examples/core.flux](examples/core.flux) for returned closures and local
recursion. Execution allows 256 active function calls by default and reports a
located error when the limit is exceeded. Phase 3's breaking changes and
bytecode format 2 are described in [the migration notes](docs/MIGRATION.md).

## Configuration System

Flux uses a `flux.json` configuration file to control type checking behavior and compiler settings. You can create one using:

```bash
./flux init
```

### Configuration Options

The `flux.json` file supports the following options:

```json
{
  "typeChecking": {
    "strict": false,        // Enable strict type checking
    "warnOnly": false,      // Convert type errors to warnings
    "enabled": true         // Enable/disable type checking entirely
  },
  "compiler": {
    "optimizationLevel": 1, // Reserved for future optimization support
    "debug": false          // Embed source text for diagnostic snippets
  }
}
```

### Type Checking Modes

#### 1. **Disabled** (`enabled: false`)
- No type checking performed
- Fastest compilation
- Runtime type errors possible

```json
{
  "typeChecking": {
    "enabled": false
  }
}
```

#### 2. **Lenient** (`strict: false, warnOnly: false`)
- **Default mode**
- Type checking with some flexibility
- Differing branch types and unlike-type equality issue warnings
- Conditions require `bool`; binding errors remain errors in every mode
- Invalid arithmetic and annotated assignments remain errors; no implicit conversion is performed
- Good for gradual adoption

```json
{
  "typeChecking": {
    "strict": false,
    "warnOnly": false,
    "enabled": true
  }
}
```

#### 3. **Warn-Only** (`warnOnly: true`)
- All type errors become warnings
- Code still executes even with type issues
- Good for migration from untyped code

```json
{
  "typeChecking": {
    "warnOnly": true,
    "enabled": true
  }
}
```

#### 4. **Strict** (`strict: true`)
- Maximum type safety
- No implicit conversions
- All type mismatches are errors
- Use explicit function signatures for the strongest checking

```json
{
  "typeChecking": {
    "strict": true,
    "warnOnly": false,
    "enabled": true
  }
}
```

## Type System

Flux checks concrete types and annotations before execution. An unannotated
function parameter is given an unknown type that is compatible with everything,
so it is not checked; inference is not a constraint solver. The rules are
`TYP-*` in [the specification](docs/SPEC.md#4-types-and-annotations).

### Basic Types
- `int`: Integer numbers
- `string`: Text strings  
- `bool`: Boolean values (true/false)
- `void`: No value

### Composite Types
- `[T]`: Lists of type T (e.g., `[int]`, `[string]`)
- `{K: V}`: Dictionaries with key type K and value type V (e.g., `{string: int}`)
- `fn(T1, T2, ...) -> R`: Function types with parameter types and return type

### Type Annotations

You can add optional type annotations to variables and function parameters:

```flux
// Variable type annotations
let x: int = 42
let name: string = "Flux"
let active: bool = true

// Function with typed parameters and return type
let add: fn(int, int) -> int = fn(a: int, b: int): int => a + b

// Lists and dictionaries with type annotations
let numbers: [int] = [1, 2, 3, 4, 5]
let person: {string: string} = {"name": "Alice", "city": "Tokyo"}
```

## Installation

```bash
# Clone the repository
git clone https://github.com/pranavms13/flux-lang.git
cd flux-lang

# Build the project
go build -o dist/flux
```

## Usage

#### Initialize a new project:

```bash
./dist/flux init
```

This creates a `flux.json` configuration file with sensible defaults and refuses to overwrite an existing file. Configuration is read from the current working directory.

#### To run a Flux program:

```bash
./dist/flux run <filename>
```

For example:
```bash
./dist/flux run main.flux
```

#### To compile a Flux program to a binary:

Compilation requires Go on `PATH`, and writes the executable to `dist/`. The generated executable runs independently of Go and the Flux source checkout.

```bash
./dist/flux compile <filename>
```

For example:
```bash
./dist/flux compile main.flux
```

## Example Code

Here's a simple example of Flux code without type annotations (backward compatible):

```flux
print("Functions")
let double = fn(x) => x + x
let result = double(5)
print(result)

print("Add Strings")
let name = "Flux"
print("Hello, " + name)

let x = 5
let msg = if x > 0 then {
  print("x is positive")
  "yes"
} else {
  "no"
}
print(msg)
```

Here's an example with type safety features:

```flux
// Basic typed variable declarations
let x: int = 42
let name: string = "Flux"
let isActive: bool = true

// Typed function declarations
let add: fn(int, int) -> int = fn(a: int, b: int): int => a + b
let result: int = add(10, 20)
print("Add result:")
print(result)

// List with type annotation
let numbers: [int] = [1, 2, 3, 4, 5]
print("First number:")
print(numbers[0])

// Dictionary with type annotation
let person: {string: string} = {"name": "Alice", "city": "Tokyo"}
print("Person name:")
print(person["name"])

// Function with typed parameters and return type
let greet = fn(name: string): string => "Hello, " + name
let greeting: string = greet("World")
print(greeting)
```

## Type Checking Examples

The diagnostic excerpts below are abbreviated.

### Strict Mode
```bash
# Set strict: true in flux.json
./dist/flux run examples/type_errors.flux
```
```
Error: type checking failed:
  - type mismatch: variable x declared as int but assigned string
  - if branches must have same type: then=int, else=string
```

### Lenient Mode
```bash
# Set strict: false in flux.json  
./dist/flux run examples/type_errors.flux
```
```
Type checking warnings:
  - if branches must have same type: then=int, else=string (using unknown type)
Error: type checking failed:
  - type mismatch: variable x declared as int but assigned string
```

### Warn-Only Mode
```bash
# Set warnOnly: true in flux.json
./dist/flux run examples/type_errors.flux
```
```
Type checking warnings:
  - type mismatch: variable x declared as int but assigned string
  - if branches must have same type: then=int, else=string (using unknown type)
# Code attempts to execute...
```

For more examples, look into [Examples](./examples). `type_errors.flux` is intentionally invalid; the other examples run with the default configuration.

## Diagnostics

Every source-related language diagnostic names the file, the line and the
construct it is about, carries a stable code, and — where it helps — points at
the declaration it disagrees with.

A type error shows both halves of the disagreement, because knowing which of the
two is wrong is the actual question:

```text
types.flux:2:11: error[T_ARGUMENT_TYPE]: argument 1 has type string, expected int
2 | print(add("5", 10))
  |           ^^^
  = note: parameter "a" is declared as int here at types.flux:1:14
```

A failure while running says which calls led there:

```text
trace.flux:1:24: error[R_OPERAND_TYPE]: cannot apply + to int and string
1 | let inner = fn(x) => x + "no"
  |                        ^^^^^^
  in inner, called at trace.flux:2:27
  in outer, called at trace.flux:3:12
```

An unfinished construct is reported as one, rather than as an unexpected token
that happens to be the end of the file:

```text
syntax.flux:2:1: error[S_UNEXPECTED_EOF]: unexpected end of input, expected Expr
  = note: a construct started earlier in the file was never finished
```

Compiled executables report the same way. Their positions are resolved when the
program is compiled, not looked up when it fails, so a built program still says
`main.flux:2:9` after the `.flux` file is gone. Set `compiler.debug` in
`flux.json` to embed the source as well, and the executable can show the
offending line too — at the cost of a larger binary that contains your source.

Diagnostics go to stderr and program output to stdout, so a pipeline reads one
without the other. Commands exit `0` on success, `1` when a program is rejected
or fails while running, and `2` when Flux could not do the job at all — bad
usage, a missing file, or a bug in Flux.

Colour is used only when writing to a terminal, and never when `NO_COLOR` is
set, so redirected output is byte-stable.

The code in brackets is the stable part; wording may improve, codes do not
change meaning. They are listed in
[the diagnostic reference](docs/DIAGNOSTICS.md).

## Configuration Examples

See the `examples/` directory for configuration examples:
- `flux.strict.json` - Strict type checking configuration
- `flux.lenient.json` - Lenient configuration for gradual adoption

## VS Code Extension
To use the Flux Language extension in VS Code:

1. Install the dependencies:
   ```bash
    npm install -g yo generator-code
   ```

2. Build from source:
   ```bash
   cd vsce
   vsce package
   ```
   This will create a `.vsix` file in the `vsce` directory.

3. Install VSIX File to IDE:
   - Open VS Code / Compatible IDE
   - Open Settings Menu (Ctrl+Shift+P)
   - Click on the "Extensions: Install from VSIX..." option.
   - Navigate to and select the `.vsix` file created in the previous step
   - Reload VS Code if prompted

The extension provides:
- Syntax highlighting for `.flux` files
- Basic language support
- Bracket matching
- Comment toggling


## Development

See [the development baseline](docs/DEVELOPMENT.md) for the execution pipeline, validation commands, current limitations, and next priorities.

```bash
make check
make test-coverage
make build-all
```

`+` and `-` associate left to right and bind more tightly than `==`, `<`, and `>`. Use parentheses to group expressions. `print(value)` outputs immediately and returns void.

## Performance

Phase 3 measurements on an Apple M3 Pro (darwin/arm64, Go 1.25.1), using
100 ms benchmark windows:

| Stage | small | nested | large |
| --- | --- | --- | --- |
| Parse | 504 µs | 5.71 ms | 13.2 ms |
| Resolve and type check | 11.2 µs | 18.0 µs | 401 µs |

`small` is a seven-line script, `nested` has six conditional levels and a
closure, and `large` has 300 bindings and a 300-element list. These are local
measurements, not performance guarantees. The larger grammar and binding
analysis add work compared with the earlier baseline.

Dictionary/block ambiguity still causes exponential work when parsing nested
conditionals. Keep nesting shallow until that parser limitation is addressed.
See [the Phase 3 report](docs/PHASE3.md) for allocations and methodology, and
[the historical baseline](docs/BASELINE.md) for earlier measurements.

## Project Structure

- `source/` - Source snapshots and the byte spans that locate code in them
- `render/` - The one place a diagnostic becomes text: snippets, colour, JSON
- `fault/` - The catalogue of failures a program can produce while running
- `diagnostic/` - Structured error reporting: codes, severities, spans, notes
- `lexer/` - Tokenizes source code into tokens
- `parser/` - Parses tokens into an Abstract Syntax Tree (AST)
- `resolver/` - Lexical binding identities, scopes and captures
- `value/` - Checked arithmetic, equality, indexing and display
- `types/` - Type system implementation with type checking
- `config/` - Configuration system for flux.json
- `compiler/` - Compiles AST into bytecode
- `vm/` - Virtual machine that executes bytecode
- `ast/` - Core AST node definitions with type annotation support
- `runtime/` - Runtime functionality and built-in functions
- `internal/conformance/` - The fixture format the specification is checked with
- `internal/fixtures/` - What every file under `examples/` is expected to do
- `testdata/conformance/` - One or more fixtures for every rule in the specification
- `docs/` - The [specification](docs/SPEC.md), the [decisions](docs/decisions/) behind
  it, the [migration notes](docs/MIGRATION.md), the [diagnostic codes](docs/DIAGNOSTICS.md),
  and the [plan](docs/PLAN.md)
- `vsce/` - VS Code Extension for Flux Language

## Dependencies

- Go 1.23.2 or higher
- github.com/alecthomas/participle/v2 - For parsing

## License

MIT License

Copyright (c) 2025 Pranav M S

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 
