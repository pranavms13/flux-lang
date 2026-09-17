package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/config"
	"github.com/pranavms13/flux-lang/internal/fixtures"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

// TestFixtureManifestCoversExamples is the check that keeps the manifest
// honest: a fixture added to examples/ without a declaration, or a declaration
// left behind after its file is deleted, fails here rather than quietly
// dropping out of the suite.
func TestFixtureManifestCoversExamples(t *testing.T) {
	entries, err := os.ReadDir("examples")
	if err != nil {
		t.Fatal(err)
	}
	declared := make(map[string]bool, len(fixtures.Manifest))
	for _, fixture := range fixtures.Manifest {
		if declared[fixture.Path] {
			t.Errorf("%s is declared twice", fixture.Path)
		}
		declared[fixture.Path] = true
		if _, err := os.Stat(fixture.Path); err != nil {
			t.Errorf("%s is declared but missing: %v", fixture.Path, err)
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.ToSlash(filepath.Join("examples", entry.Name()))
		if !declared[path] {
			t.Errorf("%s is not declared in the fixture manifest; say what it proves "+
				"and what it is expected to do in each type-checking mode", path)
		}
	}
}

func TestFixtureManifestIsCoherent(t *testing.T) {
	for _, fixture := range fixtures.Manifest {
		t.Run(fixture.Name(), func(t *testing.T) {
			if err := fixture.Validate(); err != nil {
				t.Error(err)
			}
		})
	}
	kinds := map[fixtures.Kind]int{}
	for _, fixture := range fixtures.Manifest {
		kinds[fixture.Kind()]++
	}
	// The manifest is only worth having if it separates kinds that a filename
	// cannot. Losing the last mode-dependent fixture would make it decorative.
	for _, kind := range []fixtures.Kind{fixtures.KindValid, fixtures.KindModeDependent, fixtures.KindConfig} {
		if kinds[kind] == 0 {
			t.Errorf("no fixture of kind %q remains", kind)
		}
	}
}

func TestFixtureConfigsSelectTheDeclaredMode(t *testing.T) {
	for _, fixture := range fixtures.Configs() {
		t.Run(fixture.Name(), func(t *testing.T) {
			contents, err := os.ReadFile(fixture.Path)
			if err != nil {
				t.Fatal(err)
			}
			loaded := config.DefaultConfig()
			if err := json.Unmarshal(contents, loaded); err != nil {
				t.Fatalf("parse configuration: %v", err)
			}
			want := fixture.Selects.TypeChecking()
			got := types.TypeCheckingMode{
				Strict:   loaded.TypeChecking.Strict,
				WarnOnly: loaded.TypeChecking.WarnOnly,
				Enabled:  loaded.TypeChecking.Enabled,
			}
			if got != want {
				t.Errorf("%s configures %+v, but the manifest says it selects %q (%+v)",
					fixture.Path, got, fixture.Selects, want)
			}
		})
	}
}

// TestFixturePrograms runs every declared program in every mode on both
// backends and checks the declared outcome. Running a valid fixture under all
// four modes is deliberate: it proves that type checking decides whether a
// program runs, never what it prints.
func TestFixturePrograms(t *testing.T) {
	for _, fixture := range fixtures.Programs() {
		source, err := os.ReadFile(fixture.Path)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			for _, mode := range fixtures.Modes {
				t.Run(string(mode), func(t *testing.T) {
					for _, backend := range []string{"interpreter", "vm"} {
						t.Run(backend, func(t *testing.T) {
							assertOutcome(t, fixture, mode, backend, string(source))
						})
					}
				})
			}
		})
	}
}

func assertOutcome(t *testing.T, fixture fixtures.Fixture, mode fixtures.Mode, backend, text string) {
	t.Helper()
	want := fixture.Outcome(mode)

	result := parser.ParseSource(source.New(1, fixture.Path, text))
	if result.Failed() {
		assertStatic(t, want, fixture, mode, []string{result.Diagnostics[0].Message})
		return
	}
	prog, src := result.Program, result.Source
	checker := types.NewTypeCheckerForSource(src, mode.TypeChecking())
	checker.CheckProgram(prog)
	if checker.HasErrors() {
		assertStatic(t, want, fixture, mode, checker.GetErrors())
		return
	}

	output, failure := execute(prog, src, backend)
	switch {
	case failure != nil && want != fixtures.OutcomeRuntimeError:
		t.Errorf("%s in %s mode: failed at run time (%v), but the manifest expects %q",
			fixture.Path, mode, failure, want)
	case failure == nil && want == fixtures.OutcomeRuntimeError:
		t.Errorf("%s in %s mode: ran to completion, but the manifest expects it to fail at run time",
			fixture.Path, mode)
	case failure != nil:
		return
	case want != fixtures.OutcomeOutput:
		t.Errorf("%s in %s mode: ran to completion, but the manifest expects %q", fixture.Path, mode, want)
	case output != fixture.Stdout:
		t.Errorf("%s in %s mode: output = %q, want %q", fixture.Path, mode, output, fixture.Stdout)
	}
}

func assertStatic(t *testing.T, want fixtures.Outcome, fixture fixtures.Fixture, mode fixtures.Mode, reported []string) {
	t.Helper()
	if want != fixtures.OutcomeStaticError {
		t.Errorf("%s in %s mode: rejected before running (%v), but the manifest expects %q",
			fixture.Path, mode, reported, want)
	}
}

// execute runs a program on one backend. Both engines return their failures, so
// a panic escaping here would be a defect in Flux rather than a mistake in the
// fixture, and is left to fail the test loudly.
func execute(prog *ast.Program, src *source.Source, backend string) (string, error) {
	var out bytes.Buffer
	if backend == "interpreter" {
		return capture(&out, runtime.Run(prog, runtime.Options{Output: &out, Source: src}))
	}
	chunk := compiler.NewFluxCompilerForSource(src).Compile(prog)
	return capture(&out, vm.NewWithOutput(chunk, &out).Run())
}

func capture(out *bytes.Buffer, err error) (string, error) { return out.String(), err }
