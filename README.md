# Flux Language

Flux is a simple, interpreted programming language implemented in Go. It features a clean syntax, **configurable static type safety**, and supports basic programming constructs like functions, conditionals, and string operations.

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/pranavms13/flux-lang)
## Author

**Pranav M S**
- GitHub: [@pranavms13](https://github.com/pranavms13)
- Email: [flux@pranavms.dev](mailto:flux@pranavms.dev)

## Supported Features

- **Configurable Type Safety**: Static type checking with multiple modes via `flux.json`
- Basic and advanced type annotations (int, string, bool, void, lists, dictionaries, functions)
- Basic Inline Function definitions and calls with typed parameters
- Strings with type checking
- Conditional expressions with type validation
- Basic arithmetic operations with type safety
- Print statements for output
- Lists with homogeneous type checking
- Dictionaries with typed keys and values
- Type inference for backward compatibility

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
    "optimizationLevel": 1, // Compilation optimization level (0-3)
    "debug": false          // Enable debug information
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
- Mixed-type operations issue warnings but may be allowed
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
- Recommended for production code

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

Flux includes a comprehensive type system that provides compile-time type safety:

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

This creates a `flux.json` configuration file with sensible defaults.

#### To run a Flux program:

```bash
./dist/flux run <filename>
```

For example:
```bash
./dist/flux run main.flux
```

#### To compile a Flux program to a binary:

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

### Strict Mode
```bash
# Set strict: true in flux.json
./dist/flux run examples/type_errors.flux
```
```
Type checking errors:
  - type mismatch: variable x declared as int but assigned string
  - if branches must have same type: then=int, else=string
Execution failed due to type errors.
```

### Lenient Mode
```bash
# Set strict: false in flux.json  
./dist/flux run examples/type_errors.flux
```
```
Type checking warnings:
  - if branches must have same type: then=int, else=string (using union type)
Type checking errors:
  - type mismatch: variable x declared as int but assigned string
Execution failed due to type errors.
```

### Warn-Only Mode
```bash
# Set warnOnly: true in flux.json
./dist/flux run examples/type_errors.flux
```
```
Type checking warnings:
  - type mismatch: variable x declared as int but assigned string
  - if branches must have same type: then=int, else=string (using union type)
# Code attempts to execute...
```

For more examples, look into [Examples](./examples)

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
- Code snippets
- Bracket matching
- Comment toggling


## Project Structure

- `lexer/` - Tokenizes source code into tokens
- `parser/` - Parses tokens into an Abstract Syntax Tree (AST)
- `types/` - Type system implementation with type checking
- `config/` - Configuration system for flux.json
- `compiler/` - Compiles AST into bytecode
- `vm/` - Virtual machine that executes bytecode
- `ast/` - Core AST node definitions with type annotation support
- `runtime/` - Runtime functionality and built-in functions
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
