package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/vm"
)

// invocation is what one run of the flux binary produced. The streams are kept
// apart on purpose: P1.6 requires that a program's output goes to stdout and
// every diagnostic to stderr, so a pipeline can read one without the other.
type invocation struct {
	stdout string
	stderr string
	status int
}

// TestCLI exercises built command-line binaries and standalone executables,
// including configuration, exit statuses, and output streams.
func TestCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "flux")
	if goruntime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags=-X main.Version=v1.2.3", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	dir := t.TempDir() // Deliberately outside the repository, without a go.mod.

	invoke := func(wantStatus int, args ...string) invocation {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		got := invocation{stdout: stdout.String(), stderr: stderr.String()}
		switch {
		case err == nil:
			got.status = 0
		default:
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("flux %v: %v", args, err)
			}
			got.status = exit.ExitCode()
		}
		if got.status != wantStatus {
			t.Fatalf("flux %v exited %d, want %d\nstdout: %s\nstderr: %s",
				args, got.status, wantStatus, got.stdout, got.stderr)
		}
		if strings.Contains(got.stderr, "panic:") {
			t.Fatalf("a Go panic leaked: %s", got.stderr)
		}
		return got
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	const (
		ok      = 0 // the command did what was asked
		failed  = 1 // the program was rejected or failed while running
		toolErr = 2 // bad usage, a missing file, a broken environment
	)

	if output := invoke(ok, "version"); !strings.Contains(output.stdout, "Flux Language v1.2.3\n") {
		t.Fatalf("version: %q", output.stdout)
	}
	invoke(ok, "help")

	// Mistakes in how the command was invoked are not failures of a program.
	invoke(toolErr, "unknown")
	invoke(toolErr, "run")
	invoke(toolErr, "compile")
	invoke(toolErr, "run", "missing.flux")

	invoke(ok, "init")
	write("flux.json", `{"typeChecking":{"enabled":true,"strict":true}}`)
	invoke(toolErr, "init")
	preserved, _ := os.ReadFile(filepath.Join(dir, "flux.json"))
	if string(preserved) != `{"typeChecking":{"enabled":true,"strict":true}}` {
		t.Fatal("init overwrote configuration")
	}

	write("main.flux", `
 let make = fn(x: int) => fn(y: int): int => x + y
 let add = make(10)
 let xs: [int] = [1, 2, 3]
 let d = {"a": 1, "a": 2}
 print(add(if false then { 99 } else { 3 }))
 print(xs[1]) print(d["a"])
 let log = fn(): void => print("inside")
 log()
 print(10 - 3 - 2)
 print("say \"hi\"")
 `)
	want := "13\n2\n2\ninside\n5\nsay \"hi\"\n"
	run := invoke(ok, "run", "main.flux")
	if run.stdout != want {
		t.Fatalf("run stdout: %q", run.stdout)
	}
	if run.stderr != "" {
		t.Fatalf("a successful run wrote to stderr: %q", run.stderr)
	}

	invoke(ok, "compile", "main.flux")
	executable := filepath.Join(dir, "dist", "main")
	if goruntime.GOOS == "windows" {
		executable += ".exe"
	}
	cmd := exec.Command(executable)
	cmd.Dir = t.TempDir()
	if output, err := cmd.CombinedOutput(); err != nil || string(output) != want {
		t.Fatalf("compiled: %v, %q", err, output)
	}
	// Compilation must not leave generated Go files beside executables.
	if files, _ := filepath.Glob(filepath.Join(dir, "dist", "*.go")); len(files) != 0 {
		t.Fatalf("intermediate sources: %v", files)
	}

	// A rejected program is a failure of the program, and says why on stderr.
	write("bad.flux", "let f = fn(x: int) => x\nprint(f(\"bad\"))\n")
	rejected := invoke(failed, "run", "bad.flux")
	for _, want := range []string{
		"bad.flux:2:9: error[T_ARGUMENT_TYPE]",
		"2 | print(f(\"bad\"))",
		"  |         ^^^^^",
		"= note: parameter \"x\" is declared as int here at bad.flux:1:12",
	} {
		if !strings.Contains(rejected.stderr, want) {
			t.Errorf("stderr does not contain %q:\n%s", want, rejected.stderr)
		}
	}
	if rejected.stdout != "" {
		t.Errorf("a rejected program wrote to stdout: %q", rejected.stdout)
	}
	invoke(failed, "compile", "bad.flux")

	// Warn-only downgrades the same diagnostic; the program then runs, and the
	// warning still goes to stderr rather than into the program's output.
	write("flux.json", `{"typeChecking":{"warnOnly":true}}`)
	warned := invoke(ok, "run", "bad.flux")
	if warned.stdout != "bad\n" {
		t.Errorf("warn-only stdout: %q", warned.stdout)
	}
	if !strings.Contains(warned.stderr, "warning[T_ARGUMENT_TYPE]") {
		t.Errorf("warn-only stderr: %q", warned.stderr)
	}

	write("flux.json", `{"typeChecking":{"enabled":false}}`)
	disabled := invoke(ok, "run", "bad.flux")
	if disabled.stdout != "bad\n" || disabled.stderr != "" {
		t.Errorf("disabled checking: stdout %q, stderr %q", disabled.stdout, disabled.stderr)
	}

	write("bad.flux", "let xs = [1]\nprint(xs[2])\n")
	if output := invoke(failed, "run", "bad.flux"); !strings.Contains(output.stderr, "bad.flux:2:9: error[R_INDEX_RANGE]") {
		t.Fatalf("interpreted runtime error: %q", output.stderr)
	}
	invoke(ok, "compile", "bad.flux")
	badExecutable := filepath.Join(dir, "dist", "bad")
	if goruntime.GOOS == "windows" {
		badExecutable += ".exe"
	}
	// A generated executable must report the same failure the interpreter did,
	// and must keep reporting it after the source it was built from is gone:
	// its positions were resolved at compile time, not looked up at run time.
	if err := os.Remove(filepath.Join(dir, "bad.flux")); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(badExecutable).CombinedOutput()
	switch {
	case err == nil:
		t.Fatal("the generated executable succeeded on an invalid program")
	case strings.Contains(string(output), "panic:"):
		t.Fatalf("a Go panic leaked from the generated executable: %s", output)
	case !strings.Contains(string(output), "bad.flux:2:9: error[R_INDEX_RANGE]"):
		t.Fatalf("compiled runtime error: %q", output)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != failed {
		t.Errorf("generated executable exited %v, want %d", err, failed)
	}
	// Without compiler.debug the executable has no source, so it reports the
	// position without the line rather than inventing one.
	if strings.Contains(string(output), "print(xs[2])") {
		t.Errorf("a non-debug executable embedded its source: %s", output)
	}

	write("bad.flux", `let x =`)
	if output := invoke(failed, "run", "bad.flux"); !strings.Contains(output.stderr, "error[S_UNEXPECTED_EOF]") {
		t.Fatalf("parse error: %q", output.stderr)
	}
	// A broken configuration file is an environment problem, not a program one.
	write("flux.json", `{invalid`)
	invoke(toolErr, "run", "main.flux")
	invoke(ok, "version")
}

// TestCompileDebugEmbedsSource checks the compiler.debug setting: with it, a
// generated executable can show the offending line; without it, it still
// reports the code, the position and the trace.
func TestCompileDebugEmbedsSource(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "flux")
	if goruntime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	dir := t.TempDir()
	const program = "let xs = [1]\nprint(xs[5])\n"
	if err := os.WriteFile(filepath.Join(dir, "debug.flux"), []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	config := `{"compiler":{"debug":true}}`
	if err := os.WriteFile(filepath.Join(dir, "flux.json"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	compile := exec.Command(binary, "compile", "debug.flux")
	compile.Dir = dir
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	executable := filepath.Join(dir, "dist", "debug")
	if goruntime.GOOS == "windows" {
		executable += ".exe"
	}
	// Remove the source to prove the snippet came from inside the executable.
	if err := os.Remove(filepath.Join(dir, "debug.flux")); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(executable).CombinedOutput()
	if err == nil {
		t.Fatal("the generated executable succeeded on an invalid program")
	}
	for _, want := range []string{
		"debug.flux:2:9: error[R_INDEX_RANGE]",
		"2 | print(xs[5])",
		"  |         ^^^",
	} {
		if !strings.Contains(string(output), want) {
			t.Errorf("output does not contain %q:\n%s", want, output)
		}
	}
}

// TestCompilationFailureCleansUp checks that a failed build leaves nothing
// behind. The temporary module is removed on every path out of the compiler,
// not only the successful one.
func TestCompilationFailureCleansUp(t *testing.T) {
	before := temporaryBuildDirectories(t)

	// A chunk whose constant is not serializable makes gob refuse, which fails
	// the compile after the temporary directory would otherwise be created.
	_, err := compileExecutable(&vm.Chunk{Constants: []interface{}{make(chan int)}},
		source.New(1, "unbuildable.flux", "print(1)"), false)
	if err == nil {
		t.Fatal("compiling an unserializable program succeeded")
	}

	if after := temporaryBuildDirectories(t); len(after) != len(before) {
		t.Errorf("a failed compilation left %d temporary directories behind: %v",
			len(after)-len(before), after)
	}
}

// temporaryBuildDirectories lists remaining Flux build directories so tests
// can detect cleanup failures.
func temporaryBuildDirectories(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "flux-build-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// TestRuntimeIsolationAndCompilerReuse checks that execution state does not
// leak between runs or repeated compilations.
func TestRuntimeIsolationAndCompilerReuse(t *testing.T) {
	first, _ := parser.Parse(`let secret = 42`)
	if err := runtime.Run(first, runtime.Options{Output: io.Discard}); err != nil {
		t.Fatal(err)
	}
	second, _ := parser.Parse(`print(secret)`)
	err := runtime.Run(second, runtime.Options{Output: io.Discard})
	failure, ok := err.(*fault.Error)
	if !ok || failure.Code != fault.CodeUndefinedValue {
		t.Errorf("got %v, want an undefined-value failure; the interpreter leaked globals between programs", err)
	}

	c := compiler.NewFluxCompiler()
	c.Compile(first)
	prog, _ := parser.Parse(`print(7)`)
	chunk := c.Compile(prog)
	var out bytes.Buffer
	if err := vm.NewWithOutput(chunk, &out).Run(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "7\n" {
		t.Fatalf("reused compiler output: %q", out.String())
	}
}
