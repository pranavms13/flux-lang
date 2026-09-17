package main

import (
	"bytes"
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
	"github.com/pranavms13/flux-lang/vm"
)

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
	invoke := func(ok bool, args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if (err == nil) != ok {
			t.Fatalf("flux %v: %v\n%s", args, err, output)
		}
		if strings.Contains(string(output), "panic:") {
			t.Fatalf("Go panic leaked: %s", output)
		}
		return string(output)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if output := invoke(true, "version"); !strings.Contains(output, "Flux Language v1.2.3\n") {
		t.Fatal(output)
	}
	invoke(true, "help")
	invoke(false, "unknown")
	invoke(false, "run")
	invoke(false, "compile")
	invoke(false, "run", "missing.flux")
	invoke(true, "init")
	write("flux.json", `{"typeChecking":{"enabled":true,"strict":true}}`)
	invoke(false, "init")
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
	if output := invoke(true, "run", "main.flux"); output != want {
		t.Fatalf("run: %q", output)
	}
	invoke(true, "compile", "main.flux")
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
	write("bad.flux", `let f = fn(x: int) => x print(f("bad"))`)
	invoke(false, "run", "bad.flux")
	invoke(false, "compile", "bad.flux")
	write("flux.json", `{"typeChecking":{"warnOnly":true}}`)
	if output := invoke(true, "run", "bad.flux"); !strings.Contains(output, "Type checking warnings:") || !strings.HasSuffix(output, "bad\n") {
		t.Fatal(output)
	}
	write("flux.json", `{"typeChecking":{"enabled":false}}`)
	if output := invoke(true, "run", "bad.flux"); output != "bad\n" {
		t.Fatal(output)
	}
	write("bad.flux", "let xs = [1]\nprint(xs[2])\n")
	// The interpreter reports the failure with a code and a position.
	if output := invoke(false, "run", "bad.flux"); !strings.Contains(output, "bad.flux:2:9: error[R_INDEX_RANGE]") {
		t.Fatalf("interpreted runtime error: %q", output)
	}
	invoke(true, "compile", "bad.flux")
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
	write("bad.flux", `let x =`)
	invoke(false, "run", "bad.flux")
	write("flux.json", `{invalid`)
	invoke(false, "run", "main.flux")
	invoke(true, "version")
}

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
