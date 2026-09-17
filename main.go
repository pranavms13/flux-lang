package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"text/template"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/config"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

// Version information is set by build flags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// bundledPackages are the packages a generated executable is built from.
//
// The list is explicit rather than derived from the VM's imports, so that
// adding an import the bundle does not contain fails TestBuildBundleIsClosed
// here rather than a user's build on their machine. Every package in it must
// depend only on the standard library and on other members of the bundle.
var bundledPackages = []string{"diagnostic", "fault", "source", "vm"}

// Embedding those packages makes compilation independent of the Flux source
// checkout. Test files are excluded when the bundle is written out; the embed
// patterns cannot express that.
//
//go:embed diagnostic/*.go fault/*.go source/*.go vm/*.go
var bundleFS embed.FS

func init() { gob.RegisterName("flux.Chunk", &vm.Chunk{}) }

const executableTemplate = `package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/vm"
)

func init() { gob.RegisterName("flux.Chunk", &vm.Chunk{}) }

func main() {
	bytecode, err := base64.StdEncoding.DecodeString("{{.Bytecode}}")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: unreadable embedded program:", err)
		os.Exit(2)
	}
	var chunk vm.Chunk
	if err := gob.NewDecoder(bytes.NewReader(bytecode)).Decode(&chunk); err != nil {
		fmt.Fprintln(os.Stderr, "error: unreadable embedded program:", err)
		os.Exit(2)
	}
	if err := vm.New(&chunk).Run(); err != nil {
		fmt.Fprintln(os.Stderr, fault.Report(err))
		os.Exit(1)
	}
}
`

const bundleGoMod = `module github.com/pranavms13/flux-lang

go 1.23.2
`

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runCLI(args []string) (err error) {
	// Both engines return their failures, so nothing a program does should
	// reach this. Anything that does is a defect in Flux, and is reported as
	// one: presenting it as a mistake in the user's program would send them
	// looking for a bug that is not theirs.
	defer func() {
		if failure := recover(); failure != nil {
			err = errors.New(fault.Report(&fault.Error{
				Code:    diagnostic.CodeInternal,
				Message: fmt.Sprintf("unrecovered panic: %v", failure),
			}))
		}
	}()
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		printUsage()
		return nil
	case "version", "--version":
		fmt.Printf("Flux Language v%s\nCommit: %s\nBuild Date: %s\n", strings.TrimPrefix(Version, "v"), Commit, Date)
		return nil
	case "init":
		if len(args) != 1 {
			return fmt.Errorf("usage: flux init")
		}
		if err := initializeProject(); err != nil {
			return err
		}
		fmt.Println("Initialized new Flux project with flux.json configuration file")
		return nil
	case "run", "compile":
		if len(args) != 2 {
			return fmt.Errorf("%s command requires one file argument", args[0])
		}
	default:
		return fmt.Errorf("unknown command %q; use flux help", args[0])
	}
	cfg, err := config.GetConfigFromCurrentDir()
	if err != nil {
		return err
	}
	text, err := os.ReadFile(args[1])
	if err != nil {
		return diagnostic.Tool("read source file", args[1], err)
	}
	parsed := parser.ParseSource(source.NewMap().Add(args[1], string(text)))
	if parsed.Failed() {
		return errors.New(formatDiagnostics(parsed.Source, parsed.Diagnostics))
	}
	prog := parsed.Program
	tc := types.NewTypeCheckerForSource(parsed.Source, types.TypeCheckingMode{
		Strict: cfg.TypeChecking.Strict, WarnOnly: cfg.TypeChecking.WarnOnly, Enabled: cfg.TypeChecking.Enabled,
	})
	tc.CheckProgram(prog)
	if tc.HasWarnings() {
		fmt.Fprintln(os.Stderr, "Type checking warnings:")
		for _, warning := range tc.GetWarnings() {
			fmt.Fprintf(os.Stderr, "  - %s\n", warning)
		}
	}
	if tc.HasErrors() {
		return fmt.Errorf("type checking failed:\n  - %s", strings.Join(tc.GetErrors(), "\n  - "))
	}
	if args[0] == "run" {
		if failure := runtime.Run(prog, runtime.Options{Output: os.Stdout, Source: parsed.Source}); failure != nil {
			return errors.New(fault.Report(failure))
		}
		return nil
	}
	chunk := compiler.NewFluxCompilerForSource(parsed.Source).Compile(prog)
	output, err := compileExecutable(chunk, args[1])
	if err != nil {
		return err
	}
	fmt.Printf("Compiled executable created at %s\n", output)
	return nil
}

func compileExecutable(chunk *vm.Chunk, sourcePath string) (string, error) {
	var bytecode bytes.Buffer
	if err := gob.NewEncoder(&bytecode).Encode(chunk); err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "flux-build-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)
	tmpl, err := template.New("executable").Parse(executableTemplate)
	if err != nil {
		return "", err
	}
	var generated bytes.Buffer
	if err := tmpl.Execute(&generated, map[string]string{"Bytecode": base64.StdEncoding.EncodeToString(bytecode.Bytes())}); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), generated.Bytes(), 0600); err != nil {
		return "", err
	}
	if err := writeBundle(tempDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll("dist", 0755); err != nil {
		return "", err
	}
	name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	output := filepath.Join("dist", name)
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("go", "build", "-o", absoluteOutput, ".")
	cmd.Dir = tempDir
	// The bundle is a self-contained module with no external requirements.
	// Workspace discovery is disabled so a go.work above the temporary
	// directory cannot redirect its packages, and the proxy is disabled so a
	// missing member of the bundle fails here rather than being downloaded.
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
	if result, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build executable (Go must be installed): %w\n%s", err, result)
	}
	return output, nil
}

// formatDiagnostics renders diagnostics as one line each, with their notes.
// It is a stopgap: P1.6 adds the real renderer, with source snippets, related
// locations, optional colour, and a JSON representation, and the checker and
// both engines report through it too.
func formatDiagnostics(src *source.Source, diagnostics []diagnostic.Diagnostic) string {
	rendered := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		position := src.Position(d.Primary.Start)
		line := fmt.Sprintf("%s:%d:%d: %s[%s]: %s",
			src.Name(), position.Line, position.Display, d.Severity, d.Code, d.Message)
		for _, note := range d.Notes {
			line += "\n  = note: " + note
		}
		rendered = append(rendered, line)
	}
	return strings.Join(rendered, "\n")
}

// writeBundle materializes the bundled packages under the module path they
// import each other by, alongside a minimal go.mod. The generated program then
// imports the VM the ordinary way, instead of the VM's source being rewritten
// into the main package, which only ever worked while the VM imported nothing.
func writeBundle(dir string) error {
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(bundleGoMod), 0600); err != nil {
		return err
	}
	for _, pkg := range bundledPackages {
		entries, err := bundleFS.ReadDir(pkg)
		if err != nil {
			return fmt.Errorf("bundle package %s: %w", pkg, err)
		}
		if err := os.MkdirAll(filepath.Join(dir, pkg), 0755); err != nil {
			return err
		}
		for _, entry := range entries {
			name := entry.Name()
			// Tests are not part of a program; embedding them would drag in
			// testing-only imports that the bundle does not contain.
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			contents, err := bundleFS.ReadFile(path.Join(pkg, name))
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, pkg, name), contents, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func printUsage() {
	fmt.Println("Usage: flux <command> [file.flux]")
	fmt.Println("Commands:")
	fmt.Println("\tversion - Show version information")
	fmt.Println("\tcompile <file.flux> - Compile to a standalone executable (requires Go)")
	fmt.Println("\trun <file.flux> - Run a Flux source file")
	fmt.Println("\tinit - Create a flux.json configuration file")
}

func initializeProject() error {
	f, err := os.OpenFile("flux.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("initialize project: %w", err)
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config.DefaultConfig()); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
