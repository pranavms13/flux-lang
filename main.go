package main

import (
	"bytes"
	"embed"
	"encoding/base64"
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
	"github.com/pranavms13/flux-lang/render"
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
var bundledPackages = []string{"diagnostic", "fault", "render", "source", "value", "vm"}

// Embedding those packages makes compilation independent of the Flux source
// checkout. Test files are excluded when the bundle is written out; the embed
// patterns cannot express that.
//
//go:embed diagnostic/*.go fault/*.go render/*.go source/*.go value/*.go vm/*.go
var bundleFS embed.FS

const executableTemplate = `package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/render"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/vm"
)

func main() {
	bytecode, err := base64.StdEncoding.DecodeString("{{.Bytecode}}")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: this executable's embedded program is unreadable:", err)
		os.Exit(diagnostic.ExitToolFailure)
	}
	program, err := vm.Decode(bytecode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(diagnostic.ExitToolFailure)
	}
	if failure := vm.New(program.Chunk).Run(); failure != nil {
		renderer := render.Renderer{Color: render.ColorEnabled(os.Stderr)}
		// A program built with compiler.debug carries its own source, so it can
		// show the offending line. Without it the position, code and trace are
		// still reported.
		if text, ok := program.Sources[failureFile(failure)]; ok {
			renderer.Source = source.New(1, failureFile(failure), text)
		}
		fmt.Fprintln(os.Stderr, renderer.Failure(asFault(failure)))
		os.Exit(diagnostic.ExitFailure)
	}
}

func asFault(err error) *fault.Error {
	if failure, ok := err.(*fault.Error); ok {
		return failure
	}
	return &fault.Error{Code: diagnostic.CodeInternal, Message: err.Error()}
}

func failureFile(err error) string {
	if failure, ok := err.(*fault.Error); ok {
		return failure.Where.File
	}
	return ""
}
`

const bundleGoMod = `module github.com/pranavms13/flux-lang

go 1.23.2
`

// main runs the CLI, writes failures to stderr, and exits with the status for
// that failure category.
func main() {
	// Diagnostics go to stderr; whatever the program printed has already gone
	// to stdout, so a pipeline reads output without error text mixed into it.
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, report(err))
		os.Exit(exitStatus(err))
	}
}

// usageError is a mistake in how the command was invoked, as opposed to a
// mistake in a Flux program.
type usageError struct{ message string }

// Error returns the usage message without adding a source location.
func (e *usageError) Error() string { return e.message }

// usagef formats an invalid-command error that receives the tool-failure exit
// status.
func usagef(format string, args ...any) error {
	return &usageError{message: fmt.Sprintf(format, args...)}
}

// failed marks an error as a Flux program that was rejected or that failed
// while running, which is the one outcome that is not a tool failure.
type failed struct{ error }

// exitStatus maps an error to the convention every Flux command follows.
func exitStatus(err error) int {
	var programFailure failed
	if errors.As(err, &programFailure) {
		return diagnostic.ExitFailure
	}
	return diagnostic.ExitToolFailure
}

// report renders an error for the terminal. Program failures were rendered
// where they were produced, because only there is the source available.
func report(err error) string {
	var tool *diagnostic.ToolError
	if errors.As(err, &tool) {
		return render.Renderer{Color: render.ColorEnabled(os.Stderr)}.ToolError(tool)
	}
	var programFailure failed
	if errors.As(err, &programFailure) {
		return programFailure.Error()
	}
	return "error: " + err.Error()
}

// runCLI dispatches commands through parsing, checking, and execution or
// compilation, recovering unexpected panics as internal defects.
func runCLI(args []string) (err error) {
	// Both engines return their failures, so nothing a program does should
	// reach this. Anything that does is a defect in Flux, and is reported as
	// one: presenting it as a mistake in the user's program would send them
	// looking for a bug that is not theirs.
	defer func() {
		if raised := recover(); raised != nil {
			err = errors.New(render.Renderer{Color: render.ColorEnabled(os.Stderr)}.Failure(&fault.Error{
				Code:    diagnostic.CodeInternal,
				Message: fmt.Sprintf("unrecovered panic: %v", raised),
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
			return usagef("usage: flux init")
		}
		if err := initializeProject(); err != nil {
			return err
		}
		fmt.Println("Initialized new Flux project with flux.json configuration file")
		return nil
	case "run", "compile":
		if len(args) != 2 {
			return usagef("%s command requires one file argument", args[0])
		}
	default:
		return usagef("unknown command %q; use flux help", args[0])
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
	renderer := render.Renderer{Source: parsed.Source, Color: render.ColorEnabled(os.Stderr)}
	if parsed.Failed() {
		return failed{errors.New(renderer.Diagnostics(parsed.Diagnostics))}
	}
	prog := parsed.Program
	tc := types.NewTypeCheckerForSource(parsed.Source, types.TypeCheckingMode{
		Strict: cfg.TypeChecking.Strict, WarnOnly: cfg.TypeChecking.WarnOnly, Enabled: cfg.TypeChecking.Enabled,
	})
	tc.CheckProgram(prog)
	if warnings := tc.DiagnosticsWithSeverity(diagnostic.SeverityWarning); len(warnings) > 0 {
		fmt.Fprintln(os.Stderr, renderer.Diagnostics(warnings))
	}
	if errs := tc.DiagnosticsWithSeverity(diagnostic.SeverityError); len(errs) > 0 {
		return failed{errors.New(renderer.Diagnostics(errs))}
	}
	if args[0] == "run" {
		failure := runtime.Run(prog, runtime.Options{Output: os.Stdout, Source: parsed.Source})
		if failure == nil {
			return nil
		}
		reported, ok := failure.(*fault.Error)
		if !ok {
			return failure
		}
		// An internal defect is not a failure of the user's program, so it does
		// not get the status that says one failed.
		if reported.Code == diagnostic.CodeInternal {
			return errors.New(renderer.Failure(reported))
		}
		return failed{errors.New(renderer.Failure(reported))}
	}
	chunk := compiler.NewFluxCompilerForSource(parsed.Source).Compile(prog)
	output, err := compileExecutable(chunk, parsed.Source, cfg.Compiler.Debug)
	if err != nil {
		return err
	}
	fmt.Printf("Compiled executable created at %s\n", output)
	return nil
}

// compileExecutable builds a standalone VM executable in dist, optionally
// embeds source text, and always removes its temporary build module.
func compileExecutable(chunk *vm.Chunk, src *source.Source, debug bool) (string, error) {
	var sources map[string]string
	if debug {
		sources = map[string]string{src.Name(): src.Text()}
	}
	bytecode, err := vm.Encode(chunk, sources)
	if err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "flux-build-*")
	if err != nil {
		return "", err
	}
	// The temporary module is removed whether the build succeeds or fails, so a
	// failed compilation leaves nothing behind to explain.
	defer os.RemoveAll(tempDir)
	tmpl, err := template.New("executable").Parse(executableTemplate)
	if err != nil {
		return "", err
	}
	var generated bytes.Buffer
	if err := tmpl.Execute(&generated, map[string]string{"Bytecode": base64.StdEncoding.EncodeToString(bytecode)}); err != nil {
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
	name := strings.TrimSuffix(filepath.Base(src.Name()), filepath.Ext(src.Name()))
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
