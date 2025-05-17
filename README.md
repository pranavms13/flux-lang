# Flux Language

Flux is a simple, interpreted programming language implemented in Go. It features a clean syntax and supports basic programming constructs like functions, conditionals, and string operations.

## Author

**Pranav M S**
- GitHub: [@pranavms13](https://github.com/pranavms13)
- Email: [flux@pranavms.dev](mailto:flux@pranavms.dev)

## Supported Features

- Basic Inline Function definitions and calls
- Strings
- Conditional expressions
- Basic arithmetic operations
- Print statements for output
- Lists
- Dictionaries

## Installation

```bash
# Clone the repository
git clone https://github.com/pranavms13/flux-lang.git
cd flux-lang

# Build the project
go build -o dist/flux
```

## Usage

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

Here's a simple example of Flux code:

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

For more examples, look into [Examples](./examples)

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
- `compiler/` - Compiles AST into bytecode
- `vm/` - Virtual machine that executes bytecode
- `types/` - Core type definitions
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
