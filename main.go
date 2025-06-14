package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/config"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

func init() {
	// Register types for gob encoding
	gob.Register(&vm.Chunk{})
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

const executableTemplate = `package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	"github.com/pranavms13/flux-lang/vm"
)

func init() {
	gob.Register(&vm.Chunk{})
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

func main() {
	// Decode the embedded bytecode
	bytecode, err := base64.StdEncoding.DecodeString("{{.Bytecode}}")
	if err != nil {
		panic(err)
	}

	var chunk vm.Chunk
	decoder := gob.NewDecoder(bytes.NewReader(bytecode))
	if err := decoder.Decode(&chunk); err != nil {
		panic(err)
	}

	// Execute the bytecode
	vm.New(&chunk).Run()
}
`

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// Load configuration
	cfg, err := config.GetConfigFromCurrentDir()
	if err != nil {
		fmt.Printf("Warning: Could not load configuration: %v\n", err)
		cfg = config.DefaultConfig()
	}

	command := os.Args[1]
	switch command {
	case "compile":
		if len(os.Args) < 3 {
			fmt.Println("Error: compile command requires a file argument")
			printUsage()
			return
		}
		source, err := os.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		// Step 1: Parse
		prog, err := parser.Parse(string(source))
		if err != nil {
			panic(err)
		}

		// Step 2: Type Check (if enabled)
		if cfg.TypeChecking.Enabled {
			typeChecker := types.NewTypeCheckerWithConfig(types.TypeCheckingMode{
				Strict:   cfg.TypeChecking.Strict,
				WarnOnly: cfg.TypeChecking.WarnOnly,
				Enabled:  cfg.TypeChecking.Enabled,
			})

			typeChecker.CheckProgram(prog)

			// Display warnings if any
			if typeChecker.HasWarnings() {
				fmt.Println("Type checking warnings:")
				for _, warning := range typeChecker.GetWarnings() {
					fmt.Printf("  - %s\n", warning)
				}
			}

			// Handle errors
			if typeChecker.HasErrors() {
				fmt.Println("Type checking errors:")
				for _, error := range typeChecker.GetErrors() {
					fmt.Printf("  - %s\n", error)
				}
				fmt.Println("Compilation failed due to type errors.")
				os.Exit(1)
			}
		}

		// Step 3: Compile to bytecode
		chunk := compiler.NewFluxCompiler().Compile(prog)

		// Step 4: Create temporary file for bytecode
		tempFile, err := os.CreateTemp("", "flux-bytecode-*.gob")
		if err != nil {
			panic(err)
		}
		defer os.Remove(tempFile.Name())

		// Encode bytecode to temporary file
		encoder := gob.NewEncoder(tempFile)
		if err := encoder.Encode(chunk); err != nil {
			panic(err)
		}
		tempFile.Close()

		// Read the encoded bytecode
		bytecode, err := os.ReadFile(tempFile.Name())
		if err != nil {
			panic(err)
		}

		// Base64 encode the bytecode
		base64Bytecode := base64.StdEncoding.EncodeToString(bytecode)

		// Create output directory if it doesn't exist
		outputDir := "dist"
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(err)
		}

		// Create the executable source file
		baseName := filepath.Base(os.Args[2])
		execName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		execSource := filepath.Join(outputDir, execName+".go")

		// Create and write the executable source
		tmpl, err := template.New("executable").Parse(executableTemplate)
		if err != nil {
			panic(err)
		}

		execFile, err := os.Create(execSource)
		if err != nil {
			panic(err)
		}
		defer execFile.Close()

		if err := tmpl.Execute(execFile, map[string]string{
			"Bytecode": base64Bytecode,
		}); err != nil {
			panic(err)
		}

		// Build the executable
		cmd := exec.Command("go", "build", "-o", filepath.Join(outputDir, execName), execSource)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Build error: %v\n", err)
			fmt.Printf("stdout: %s\n", stdout.String())
			fmt.Printf("stderr: %s\n", stderr.String())
			panic(err)
		}

		// Remove the intermediate .go source file
		if err := os.Remove(execSource); err != nil {
			fmt.Printf("Warning: Could not remove intermediate source file: %v\n", err)
		}

		fmt.Printf("Compiled executable created at %s\n", filepath.Join(outputDir, execName))

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Error: run command requires a file argument")
			printUsage()
			return
		}
		source, err := os.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		// Step 1: Parse
		prog, err := parser.Parse(string(source))
		if err != nil {
			panic(err)
		}

		// Step 2: Type Check (if enabled)
		if cfg.TypeChecking.Enabled {
			typeChecker := types.NewTypeCheckerWithConfig(types.TypeCheckingMode{
				Strict:   cfg.TypeChecking.Strict,
				WarnOnly: cfg.TypeChecking.WarnOnly,
				Enabled:  cfg.TypeChecking.Enabled,
			})

			typeChecker.CheckProgram(prog)

			// Display warnings if any
			if typeChecker.HasWarnings() {
				fmt.Println("Type checking warnings:")
				for _, warning := range typeChecker.GetWarnings() {
					fmt.Printf("  - %s\n", warning)
				}
			}

			// Handle errors
			if typeChecker.HasErrors() {
				fmt.Println("Type checking errors:")
				for _, error := range typeChecker.GetErrors() {
					fmt.Printf("  - %s\n", error)
				}
				fmt.Println("Execution failed due to type errors.")
				os.Exit(1)
			}
		}

		// Step 3: Run
		runtime.Run(prog)
	case "init":
		// Initialize a new Flux project with default configuration
		if err := initializeProject(); err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Initialized new Flux project with flux.json configuration file")
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: flux <command> <file>.flux")
	fmt.Println("Commands:")
	fmt.Println("\tcompile <file>.flux - Compile the given Flux source file to an executable")
	fmt.Println("\trun <file>.flux - Run the given Flux source file")
	fmt.Println("\tinit - Initialize a new Flux project with default configuration")
}

func initializeProject() error {
	cfg := config.DefaultConfig()
	return config.SaveConfig(cfg, ".")
}
