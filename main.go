package main

import (
	"fmt"
	"os"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/vm"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: flux <command> <file>.flux")
		return
	}

	command := os.Args[1]
	switch command {
	case "compile":
		source, err := os.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		// Step 1: Parse
		prog, err := parser.Parse(string(source))
		if err != nil {
			panic(err)
		}

		// Step 2: Compile to bytecode
		chunk := compiler.NewFluxCompiler().Compile(prog)

		// Step 3: Execute
		vm.New(chunk).Run()
	case "run":
		source, err := os.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		// Step 1: Parse
		prog, err := parser.Parse(string(source))
		if err != nil {
			panic(err)
		}

		// Step 2: Run
		runtime.Run(prog)
	default:
		fmt.Println("Usage: flux <command> <file>.flux")
	}
}
