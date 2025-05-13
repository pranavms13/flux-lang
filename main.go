package main

import (
	"fmt"
	"os"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: flux <file>")
		return
	}

	source, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	// Step 1: Parse
	prog, err := parser.Parse(string(source))
	if err != nil {
		panic(err)
	}

	// Step 2: Compile to bytecode
	chunk := compiler.New().Compile(prog)

	// Step 3: Execute
	vm.New(chunk).Run()
}
