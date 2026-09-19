// Package fixtures declares what every file under examples/ is for.
//
// The manifest exists because a fixture's expected outcome is not visible in
// its name. examples/type_errors.flux is rejected by the checker in two modes
// and fails at run time in the other two; examples/flux.lenient.json is named
// "lenient" but configures warn-only checking. A test that guessed from either
// filename would assert the wrong thing and still pass.
//
// So every fixture states its expected outcome in each type-checking mode, and
// its kind is derived from those outcomes rather than declared independently.
// A test walks examples/ and fails on any file the manifest does not declare,
// which keeps a new fixture from being added without saying what it proves.
package fixtures

import (
	"fmt"
	"strings"

	"github.com/pranavms13/flux-lang/types"
)

// Mode names a type-checking configuration a fixture can be run under.
type Mode string

const (
	// ModeStrict reports every mismatch as an error.
	ModeStrict Mode = "strict"
	// ModeLenient is the default: mismatches are errors, but some
	// unresolved types are tolerated as warnings.
	ModeLenient Mode = "lenient"
	// ModeWarnOnly downgrades every type error to a warning and runs anyway.
	ModeWarnOnly Mode = "warn-only"
	// ModeDisabled skips type checking entirely.
	ModeDisabled Mode = "disabled"
)

// Modes lists every mode a program fixture must declare an outcome for.
var Modes = []Mode{ModeStrict, ModeLenient, ModeWarnOnly, ModeDisabled}

// TypeChecking returns the checker configuration this mode names.
func (m Mode) TypeChecking() types.TypeCheckingMode {
	switch m {
	case ModeStrict:
		return types.TypeCheckingMode{Strict: true, Enabled: true}
	case ModeLenient:
		return types.TypeCheckingMode{Enabled: true}
	case ModeWarnOnly:
		return types.TypeCheckingMode{WarnOnly: true, Enabled: true}
	case ModeDisabled:
		return types.TypeCheckingMode{}
	default:
		panic(fmt.Sprintf("fixtures: unknown mode %q", m))
	}
}

// Outcome is what running a fixture in one mode is expected to produce.
type Outcome string

const (
	// OutcomeOutput means the program is accepted and runs to completion,
	// printing the fixture's declared Stdout.
	OutcomeOutput Outcome = "output"
	// OutcomeStaticError means the program is rejected before it runs, by
	// either the parser or the checker.
	OutcomeStaticError Outcome = "static-error"
	// OutcomeRuntimeError means the program is accepted and then fails while
	// running. Warnings do not change this: a warned-about program still runs.
	OutcomeRuntimeError Outcome = "runtime-error"
)

// Kind classifies a fixture by how its outcomes vary across modes. It is
// derived, never declared, so it cannot disagree with the behavior a test
// actually checks.
type Kind string

const (
	// KindValid runs to completion in every mode.
	KindValid Kind = "valid"
	// KindStaticError is rejected in every mode.
	KindStaticError Kind = "static-error"
	// KindRuntimeError is accepted in every mode and fails while running.
	KindRuntimeError Kind = "runtime-error"
	// KindModeDependent behaves differently depending on the mode, which is
	// what makes it worth keeping as a fixture.
	KindModeDependent Kind = "mode-dependent"
	// KindConfig is not a program. It is a configuration file that selects a
	// mode for the programs run beside it.
	KindConfig Kind = "config"
)

// Fixture describes one file under examples/.
type Fixture struct {
	// Path is relative to the repository root.
	Path string
	// Purpose says what this fixture proves. A fixture nothing asserts on is
	// a file waiting to rot.
	Purpose string
	// Outcomes gives the expected result in every mode of [Modes]. It is empty
	// for a configuration fixture.
	Outcomes map[Mode]Outcome
	// Stdout is the output expected wherever Outcomes is OutcomeOutput. A
	// fixture that produces output in several modes must produce the same
	// output in all of them: type checking decides whether a program runs, not
	// what it prints.
	Stdout string
	// Selects is the mode a configuration fixture chooses. It is empty for a
	// program fixture.
	Selects Mode
}

// IsProgram reports whether the fixture is Flux source.
func (f Fixture) IsProgram() bool { return strings.HasSuffix(f.Path, ".flux") }

// Outcome returns the expected result in one mode.
func (f Fixture) Outcome(mode Mode) Outcome { return f.Outcomes[mode] }

// Kind derives the fixture's classification from its declared outcomes.
func (f Fixture) Kind() Kind {
	if !f.IsProgram() {
		return KindConfig
	}
	first, varies := f.Outcomes[Modes[0]], false
	for _, mode := range Modes[1:] {
		if f.Outcomes[mode] != first {
			varies = true
		}
	}
	switch {
	case varies:
		return KindModeDependent
	case first == OutcomeOutput:
		return KindValid
	case first == OutcomeStaticError:
		return KindStaticError
	default:
		return KindRuntimeError
	}
}

// Validate reports whether the declaration is internally coherent. It catches
// the mistakes that would otherwise make a fixture test vacuous: a program that
// declares no outcome for a mode, or one expected to produce output without
// saying what.
func (f Fixture) Validate() error {
	if f.Path == "" || f.Purpose == "" {
		return fmt.Errorf("fixture %q: needs a path and a purpose", f.Path)
	}
	if !f.IsProgram() {
		switch {
		case len(f.Outcomes) != 0 || f.Stdout != "":
			return fmt.Errorf("fixture %q: a configuration file neither runs nor prints", f.Path)
		case f.Selects == "":
			return fmt.Errorf("fixture %q: must declare the mode it selects", f.Path)
		}
		return nil
	}
	if f.Selects != "" {
		return fmt.Errorf("fixture %q: a program does not select a mode", f.Path)
	}
	producesOutput := false
	for _, mode := range Modes {
		outcome, declared := f.Outcomes[mode]
		if !declared {
			return fmt.Errorf("fixture %q: no outcome declared for mode %q", f.Path, mode)
		}
		switch outcome {
		case OutcomeOutput:
			producesOutput = true
		case OutcomeStaticError, OutcomeRuntimeError:
		default:
			return fmt.Errorf("fixture %q: unknown outcome %q for mode %q", f.Path, outcome, mode)
		}
	}
	if len(f.Outcomes) != len(Modes) {
		return fmt.Errorf("fixture %q: declares an outcome for a mode that does not exist", f.Path)
	}
	if producesOutput == (f.Stdout == "") {
		return fmt.Errorf("fixture %q: expected output and declared Stdout must agree", f.Path)
	}
	return nil
}

// allModes is shorthand for a program that behaves the same way everywhere.
func allModes(outcome Outcome) map[Mode]Outcome {
	outcomes := make(map[Mode]Outcome, len(Modes))
	for _, mode := range Modes {
		outcomes[mode] = outcome
	}
	return outcomes
}

// Manifest declares every file under examples/. Adding a file there without
// adding it here fails TestFixtureManifestCoversExamples.
var Manifest = []Fixture{
	{Path: "examples/core.flux", Purpose: "Phase 3 arithmetic, short circuiting, lexical captures, and global/local recursion", Outcomes: allModes(OutcomeOutput), Stdout: "120\nfalse\n16\n[7, -1, -9223372036854775808]\n1\n"},
	{
		Path:     "examples/main.flux",
		Purpose:  "the introductory tour: functions, string concatenation, conditionals, booleans",
		Outcomes: allModes(OutcomeOutput),
		Stdout:   "Functions\n10\nAdd Strings\nHello, Flux\nx is positive\nyes\n",
	},
	{
		Path:     "examples/typed.flux",
		Purpose:  "type annotations on bindings, parameters, return types, lists and dictionaries",
		Outcomes: allModes(OutcomeOutput),
		Stdout:   "Basic types:\n42\nFlux\nAdd result:\n30\nNumbers list:\n1\nPerson name:\nAlice\nHello, World\nStatus:\npositive\n",
	},
	{
		Path:     "examples/array.flux",
		Purpose:  "list literals and indexing, including a list of mixed element types",
		Outcomes: allModes(OutcomeOutput),
		Stdout:   "1\n5\nhello\n",
	},
	{
		Path:     "examples/dict.flux",
		Purpose:  "dictionary literals, string and int keys, and nested access",
		Outcomes: allModes(OutcomeOutput),
		Stdout:   "John\n30\nNew York\n42\nHello\ntrue\nAlice\nLondon\n",
	},
	{
		Path: "examples/type_errors.flux",
		Purpose: "every mismatch the checker is expected to catch; also proves that " +
			"downgrading those errors does not make the program work, because the " +
			"mismatches it describes are real",
		Outcomes: map[Mode]Outcome{
			ModeStrict:  OutcomeStaticError,
			ModeLenient: OutcomeStaticError,
			// Warn-only and disabled let the program run, and it fails on the
			// first mismatch the checker was warning about.
			ModeWarnOnly: OutcomeRuntimeError,
			ModeDisabled: OutcomeRuntimeError,
		},
	},
	{
		Path:    "examples/flux.strict.json",
		Purpose: "a configuration that turns strict checking on",
		Selects: ModeStrict,
	},
	{
		Path: "examples/flux.lenient.json",
		Purpose: "a configuration that downgrades errors to warnings; its name says " +
			"lenient but it selects warn-only, which is why the manifest states the " +
			"mode instead of inferring it",
		Selects: ModeWarnOnly,
	},
}

// Programs returns the Flux source fixtures.
func Programs() []Fixture {
	var programs []Fixture
	for _, fixture := range Manifest {
		if fixture.IsProgram() {
			programs = append(programs, fixture)
		}
	}
	return programs
}

// Configs returns the configuration fixtures.
func Configs() []Fixture {
	var configs []Fixture
	for _, fixture := range Manifest {
		if !fixture.IsProgram() {
			configs = append(configs, fixture)
		}
	}
	return configs
}

// Name returns the fixture's base filename, for use as a subtest name.
func (f Fixture) Name() string {
	if index := strings.LastIndex(f.Path, "/"); index >= 0 {
		return f.Path[index+1:]
	}
	return f.Path
}
