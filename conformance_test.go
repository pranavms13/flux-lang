package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/internal/conformance"
	"github.com/pranavms13/flux-lang/internal/fixtures"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/runtime"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/types"
	"github.com/pranavms13/flux-lang/vm"
)

const specPath = "docs/SPEC.md"

// engines names the two implementations of the language. Every conformance
// fixture runs on both, because a rule that only one of them obeys is not a
// rule.
var engines = []string{"interpreter", "vm"}

// result is what running one fixture in one mode on one backend produced.
type result struct {
	Kind   conformance.Kind
	Code   string
	Span   conformance.Span
	Stdout string
	// Warning is the first diagnostic the checker reported without failing the
	// program, or the zero value when it reported none.
	Warning conformance.Warning
	// Detail is the rendered diagnostic or failure, for a test message. It is
	// never asserted on: message wording is UNS-DIAGNOSTIC-TEXT.
	Detail string
}

// String renders a result the way a fixture header writes an outcome, so a
// failure message can be pasted into the fixture after it has been checked.
func (r result) String() string {
	if !r.Kind.IsError() {
		return string(r.Kind)
	}
	return fmt.Sprintf("%s %s at %s", r.Kind, r.Code, r.Span)
}

// loadSuite reads the conformance fixtures, failing the test rather than
// skipping when they cannot be read.
func loadSuite(t *testing.T) []conformance.Fixture {
	t.Helper()
	suite, err := conformance.Load(conformance.Root)
	if err != nil {
		t.Fatalf("load conformance suite: %v", err)
	}
	if len(suite) == 0 {
		t.Fatalf("%s contains no fixtures", conformance.Root)
	}
	return suite
}

// TestConformance runs every fixture in every mode on both backends and checks
// the declared outcome, diagnostic code, span and output.
//
// A planned fixture is run exactly like an implemented one. What it declares is
// today's behavior, so the suite keeps asserting something while the rule is
// unimplemented, and TestPlannedRulesStillDifferFromTheSpecification notices
// when the behavior changes.
func TestConformance(t *testing.T) {
	for _, fixture := range loadSuite(t) {
		t.Run(fixture.Name(), func(t *testing.T) {
			for _, mode := range fixtures.Modes {
				t.Run(string(mode), func(t *testing.T) {
					for _, backend := range engines {
						t.Run(backend, func(t *testing.T) {
							assertConformance(t, fixture, mode, backend)
						})
					}
				})
			}
		})
	}
}

// assertConformance compares one run against what the fixture declares.
func assertConformance(t *testing.T, fixture conformance.Fixture, mode fixtures.Mode, backend string) {
	t.Helper()
	want := fixture.Outcome(mode)
	got := runConformance(fixture, mode, backend)

	if declared, ok := fixture.Warnings[mode]; ok && got.Warning != declared {
		t.Errorf("first warning is %s, the fixture declares %s", got.Warning, declared)
	}
	if err := compareResult(got, want, fixture.ExpectedStdout(mode)); err != nil {
		t.Error(err)
	}
}

// compareResult checks output even when execution fails after printing a prefix.
// Planned-rule promotion uses the same comparison as present-day assertions.
func compareResult(got result, want conformance.Outcome, stdout string) error {
	if got.Kind != want.Kind {
		return fmt.Errorf("outcome is %q, the fixture declares %q\n  got:    %s\n  detail: %s",
			got.Kind, want.Kind, got, got.Detail)
	}
	if got.Stdout != stdout {
		return fmt.Errorf("stdout = %q, the fixture declares %q", got.Stdout, stdout)
	}
	if !want.Kind.IsError() {
		return nil
	}
	if got.Code != want.Code {
		return fmt.Errorf("code is %s, the fixture declares %s\n  got:    %s\n  detail: %s",
			got.Code, want.Code, got, got.Detail)
	}
	if got.Span != want.Span {
		return fmt.Errorf("span is %s, the fixture declares %s\n  detail: %s",
			got.Span, want.Span, got.Detail)
	}
	return nil
}

func TestConformanceChecksOutputBeforeFailure(t *testing.T) {
	for _, fixture := range loadSuite(t) {
		if fixture.Name() != "values/void.flux" {
			continue
		}
		for _, backend := range engines {
			got := runConformance(fixture, fixtures.ModeDisabled, backend)
			if got.Stdout != "a\n" {
				t.Fatalf("%s stdout = %q, want the prefix printed before failure", backend, got.Stdout)
			}
			want := fixture.Outcome(fixtures.ModeDisabled)
			for _, incorrect := range []string{"", "wrong\n", "a\na\n"} {
				if err := compareResult(got, want, incorrect); err == nil {
					t.Errorf("%s accepted incorrect partial output %q", backend, incorrect)
				}
			}
		}

		// The real program already has different static/runtime outcomes by mode.
		// Promotion must recognize that matrix and still check partial output.
		fixture.Specified = fixture.Outcomes
		fixture.SpecifiedStdout = fixture.Stdout
		if !satisfiesPlannedFixture(fixture) {
			t.Error("matching mode-specific outcomes were not recognized as implemented")
		}
		fixture.SpecifiedStdout = ""
		if satisfiesPlannedFixture(fixture) {
			t.Error("promotion ignored output before a runtime failure")
		}
		return
	}
	t.Fatal("missing values/void.flux fixture")
}

// TestConformanceBackendParity compares the two engines against each other
// rather than against the fixture.
//
// It is a separate assertion on purpose. Agreement between two implementations
// is not evidence that either is right, so it cannot replace the declared
// outcomes; but a rule the engines disagree about is a rule the language does
// not really have, and that disagreement is worth naming on its own.
func TestConformanceBackendParity(t *testing.T) {
	for _, fixture := range loadSuite(t) {
		t.Run(fixture.Name(), func(t *testing.T) {
			for _, mode := range fixtures.Modes {
				interpreted := runConformance(fixture, mode, "interpreter")
				compiled := runConformance(fixture, mode, "vm")
				// Only the asserted fields are compared. Detail holds rendered
				// message text, which is UNS-DIAGNOSTIC-TEXT.
				interpreted.Detail, compiled.Detail = "", ""
				if interpreted != compiled {
					t.Errorf("%s mode: the engines disagree\n  interpreter: %s stdout=%q\n  vm:          %s stdout=%q",
						mode, interpreted, interpreted.Stdout, compiled, compiled.Stdout)
				}
			}
		})
	}
}

// runConformance parses, checks and runs one fixture, reducing whatever
// happened to the fields a fixture declares.
func runConformance(fixture conformance.Fixture, mode fixtures.Mode, backend string) result {
	src := source.New(1, fixture.Path, fixture.Source)

	parsed := parser.ParseSource(src)
	if parsed.Failed() {
		return staticResult(src, parsed.Diagnostics[0])
	}

	checker := types.NewTypeCheckerForSource(src, mode.TypeChecking())
	checker.CheckProgram(parsed.Program)
	warning := firstWarning(src, checker)
	if errs := checker.DiagnosticsWithSeverity(diagnostic.SeverityError); len(errs) > 0 {
		reported := staticResult(src, errs[0])
		reported.Warning = warning
		return reported
	}

	var out bytes.Buffer
	var failure error
	if backend == "interpreter" {
		failure = runtime.Run(parsed.Program, runtime.Options{Output: &out, Source: src})
	} else {
		chunk := compiler.NewFluxCompilerForSource(src).Compile(parsed.Program)
		failure = vm.NewWithOutput(chunk, &out).Run()
	}
	if failure == nil {
		return result{Kind: conformance.KindOutput, Stdout: out.String(), Warning: warning}
	}
	reported := runtimeResult(src, failure, out.String())
	reported.Warning = warning
	return reported
}

// firstWarning reduces the first diagnostic the checker reported without
// failing the program. It is the half of MOD-SEVERITY-ONLY that outcomes alone
// cannot see: a warned-about program runs, so nothing else it does reveals
// whether the warning kept the code and span the error would have had.
func firstWarning(src *source.Source, checker *types.TypeChecker) conformance.Warning {
	warnings := checker.DiagnosticsWithSeverity(diagnostic.SeverityWarning)
	if len(warnings) == 0 {
		return conformance.Warning{}
	}
	return conformance.Warning{
		Code: string(warnings[0].Code),
		Span: spanOf(src, warnings[0].Primary.Start, warnings[0].Primary.End),
	}
}

// staticResult reduces a diagnostic reported before the program ran.
func staticResult(src *source.Source, d diagnostic.Diagnostic) result {
	return result{
		Kind:   conformance.KindStaticError,
		Code:   string(d.Code),
		Span:   spanOf(src, d.Primary.Start, d.Primary.End),
		Detail: d.Message,
	}
}

// runtimeResult reduces a failure raised while the program ran. Output printed
// before the failure is kept, because a fixture that fails partway still says
// what it managed to print.
func runtimeResult(src *source.Source, failure error, printed string) result {
	var reported *fault.Error
	if !errors.As(failure, &reported) {
		return result{Kind: conformance.KindRuntimeError, Code: "?", Stdout: printed,
			Detail: fmt.Sprintf("not a located failure: %v", failure)}
	}
	return result{
		Kind:   conformance.KindRuntimeError,
		Code:   string(reported.Code),
		Span:   spanOf(src, reported.Where.Start, reported.Where.End),
		Stdout: printed,
		Detail: reported.Error(),
	}
}

// spanOf converts a byte range into the 1-based line and display columns a
// fixture writes.
func spanOf(src *source.Source, start, end int) conformance.Span {
	from, to := src.Position(start), src.Position(end)
	return conformance.Span{
		StartLine: from.Line, StartColumn: from.Display,
		EndLine: to.Line, EndColumn: to.Display,
	}
}

// TestPlannedRulesStillDifferFromTheSpecification is what keeps an
// unimplemented rule from quietly being treated as done.
//
// A planned fixture records both what the rule requires and what Flux does
// today, and the two must differ. When an implementation makes them agree, this
// test fails and says so, which is the point: the rule is now implemented, and
// its fixture and its status in the specification have to be promoted rather
// than left describing a workaround that no longer exists.
func TestPlannedRulesStillDifferFromTheSpecification(t *testing.T) {
	planned := conformance.Planned(loadSuite(t))
	if len(planned) == 0 {
		t.Fatal("no planned fixtures remain; a specification with no unimplemented rules " +
			"should have none, and the roadmap in docs/PLAN.md still lists several")
	}
	for _, fixture := range planned {
		t.Run(fixture.Name(), func(t *testing.T) {
			if satisfiesPlannedFixture(fixture) {
				t.Errorf("%s now satisfies %s in every mode on both engines, which %s planned. "+
					"Promote the rule in %s and rewrite this fixture as implemented.",
					fixture.Path, fixture.Rule, fixture.Milestone, specPath)
			}
		})
	}
}

func satisfiesPlannedFixture(fixture conformance.Fixture) bool {
	for _, mode := range fixtures.Modes {
		for _, backend := range engines {
			got := runConformance(fixture, mode, backend)
			if compareResult(got, fixture.Specified[mode], fixture.SpecifiedOutput(mode)) != nil {
				return false
			}
		}
	}
	return true
}

// specRule is one normative rule as the specification declares it.
type specRule struct {
	ID string
	// Status is "implemented" or "planned".
	Status conformance.Status
	// Milestone is the phase that implements a planned rule, as a fixture
	// writes it: "phase-3".
	Milestone string
	// PinnedBy names a Go test that asserts the rule, for the few rules that
	// are about the command line or about the suite itself and so cannot be
	// one fixture's behavior.
	PinnedBy string
	// Line is where the rule is declared, for a test message.
	Line int
}

// ruleDeclaration matches a rule's list item in docs/SPEC.md:
//
//   - `RULE-ID` (implemented) — text
//   - `RULE-ID` (phase 3) — text
//   - `RULE-ID` (implemented) [TestName] — text
var ruleDeclaration = regexp.MustCompile(
	"^- `([A-Z][A-Z0-9-]*)` \\((implemented|phase ([0-9]+))\\)(?: \\[(Test[A-Za-z0-9_]+)\\])? —")

// readSpecRules extracts the normative rules from the specification.
func readSpecRules(t *testing.T) []specRule {
	t.Helper()
	text, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var rules []specRule
	for number, line := range strings.Split(string(text), "\n") {
		match := ruleDeclaration.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		rule := specRule{ID: match[1], Status: conformance.StatusImplemented, PinnedBy: match[4], Line: number + 1}
		if match[3] != "" {
			rule.Status, rule.Milestone = conformance.StatusPlanned, "phase-"+match[3]
		}
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		t.Fatalf("%s declares no rules; the declaration format changed without this test", specPath)
	}
	return rules
}

// TestSpecRulesHaveFixtures is the link that makes the specification
// executable: a rule with no fixture is prose, and prose drifts.
func TestSpecRulesHaveFixtures(t *testing.T) {
	suite := loadSuite(t)
	covered := map[string]int{}
	for _, fixture := range suite {
		covered[fixture.Rule]++
	}

	declared := map[string]bool{}
	for _, rule := range readSpecRules(t) {
		if declared[rule.ID] {
			t.Errorf("%s:%d: %s is declared twice", specPath, rule.Line, rule.ID)
		}
		declared[rule.ID] = true
		if covered[rule.ID] > 0 {
			if rule.PinnedBy != "" {
				t.Errorf("%s:%d: %s names both a fixture and the test %s; name only one",
					specPath, rule.Line, rule.ID, rule.PinnedBy)
			}
			continue
		}
		if rule.PinnedBy == "" {
			t.Errorf("%s:%d: %s has no fixture under %s and names no test that asserts it",
				specPath, rule.Line, rule.ID, conformance.Root)
			continue
		}
		if !testExists(t, rule.PinnedBy) {
			t.Errorf("%s:%d: %s names %s, which no test file defines",
				specPath, rule.Line, rule.ID, rule.PinnedBy)
		}
	}

	for _, fixture := range suite {
		if !declared[fixture.Rule] {
			t.Errorf("%s pins %s, which %s does not declare", fixture.Path, fixture.Rule, specPath)
		}
	}
}

// TestFixtureStatusMatchesTheSpecification checks that a fixture and the rule
// it pins agree about whether the rule is implemented, and about which phase
// implements it if not. Disagreement means one of the two was updated alone.
func TestFixtureStatusMatchesTheSpecification(t *testing.T) {
	rules := map[string]specRule{}
	for _, rule := range readSpecRules(t) {
		rules[rule.ID] = rule
	}
	for _, fixture := range loadSuite(t) {
		rule, ok := rules[fixture.Rule]
		if !ok {
			continue // TestSpecRulesHaveFixtures reports the missing rule.
		}
		if fixture.Status != rule.Status {
			t.Errorf("%s says %s is %q, %s says %q",
				fixture.Path, fixture.Rule, fixture.Status, specPath, rule.Status)
		}
		if fixture.Status == conformance.StatusPlanned && fixture.Milestone != rule.Milestone {
			t.Errorf("%s says %s lands in %q, %s says %q",
				fixture.Path, fixture.Rule, fixture.Milestone, specPath, rule.Milestone)
		}
	}
}

// TestUnspecifiedEntriesAreNotRules checks that the "intentionally
// unspecified" list stays a list of non-guarantees. An entry that grew a
// fixture is a rule, and belongs in the normative sections with a status.
func TestUnspecifiedEntriesAreNotRules(t *testing.T) {
	text, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	unspecified := regexp.MustCompile("^- `(UNS-[A-Z0-9-]*)` —")
	declared := map[string]bool{}
	for _, line := range strings.Split(string(text), "\n") {
		if match := unspecified.FindStringSubmatch(line); match != nil {
			declared[match[1]] = true
		}
	}
	if len(declared) == 0 {
		t.Fatalf("%s lists nothing as intentionally unspecified", specPath)
	}
	for _, fixture := range loadSuite(t) {
		if declared[fixture.Rule] {
			t.Errorf("%s pins %s, which %s lists as intentionally unspecified",
				fixture.Path, fixture.Rule, specPath)
		}
	}
	for _, rule := range readSpecRules(t) {
		if strings.HasPrefix(rule.ID, "UNS-") {
			t.Errorf("%s:%d: %s is normative but named as an unspecified entry",
				specPath, rule.Line, rule.ID)
		}
	}
}

// unreachableCodes are diagnostic codes no Flux source can provoke, with the
// reason. They are exempt from fixture coverage; everything else is not.
var unreachableCodes = map[diagnostic.Code]string{
	"R_UNDEFINED_VALUE": "name resolution rejects undefined names before either engine can evaluate them",
	"T_INVALID_ANNOTATION": "the grammar admits only well-formed types, so conversion " +
		"cannot fail on anything the parser accepts",
	"X_INTERNAL": "a defect in Flux itself, which a program must not be able to reach",
}

// TestConformanceCoversDiagnosticCodes checks that every code a program can
// provoke has a fixture that provokes it.
//
// A code with no fixture is a code whose location and wording nothing checks,
// which is how a diagnostic ends up underlining the wrong construct.
func TestConformanceCoversDiagnosticCodes(t *testing.T) {
	covered := map[string]bool{}
	for _, code := range conformance.Codes(loadSuite(t)) {
		covered[code] = true
	}
	var missing []string
	for _, code := range diagnostic.Registered() {
		if covered[string(code)] {
			if reason, exempt := unreachableCodes[code]; exempt {
				t.Errorf("%s is listed as unreachable (%s) but a fixture provokes it", code, reason)
			}
			continue
		}
		if _, exempt := unreachableCodes[code]; exempt {
			continue
		}
		missing = append(missing, string(code))
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("no conformance fixture provokes %s; add one, or list the code in "+
			"unreachableCodes with the reason it cannot be reached", strings.Join(missing, ", "))
	}
}

// TestImplementedFixturesNameRegisteredCodes catches a typo in a fixture's
// expected code before it becomes a fixture that asserts a code no stage can
// report.
//
// Only implemented fixtures are checked. A planned fixture's "specified" code
// is a decision about a diagnostic that does not exist yet, which is the whole
// point of writing it down.
func TestImplementedFixturesNameRegisteredCodes(t *testing.T) {
	registered := map[string]bool{}
	for _, code := range diagnostic.Registered() {
		registered[string(code)] = true
	}
	for _, fixture := range loadSuite(t) {
		for _, mode := range fixtures.Modes {
			code := fixture.Outcome(mode).Code
			if code != "" && !registered[code] {
				t.Errorf("%s expects %s in %s mode, which no stage registers",
					fixture.Path, code, mode)
			}
		}
		for mode, warning := range fixture.Warnings {
			if !registered[warning.Code] {
				t.Errorf("%s expects the warning %s in %s mode, which no stage registers",
					fixture.Path, warning.Code, mode)
			}
		}
	}
}

// TestConformanceStandaloneExecutables runs representative fixtures through the
// real compiler and the executables it produces.
//
// The in-process VM and a generated executable are different programs: the
// executable carries its own copy of the runtime and its own source map, and
// nothing else in this suite proves that what it carries still reports the same
// code at the same position.
func TestConformanceStandaloneExecutables(t *testing.T) {
	if testing.Short() {
		t.Skip("building executables invokes the Go toolchain")
	}
	suite := loadSuite(t)
	selected := representative(suite)
	if len(selected) == 0 {
		t.Fatal("no fixture is representative enough to compile")
	}

	binary := filepath.Join(t.TempDir(), "flux")
	if goruntime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build flux: %v\n%s", err, output)
	}

	for _, fixture := range selected {
		t.Run(fixture.Name(), func(t *testing.T) {
			dir := t.TempDir() // Outside the checkout, so nothing resolves through it.
			name := strings.ReplaceAll(fixture.Name(), "/", "_")
			name = strings.TrimSuffix(name, ".flux")
			if err := os.WriteFile(filepath.Join(dir, name+".flux"), []byte(fixture.Source), 0600); err != nil {
				t.Fatal(err)
			}
			// Checking is disabled so that a fixture which is rejected in other
			// modes still reaches the compiler, and debug is on so the
			// executable carries the source its diagnostics quote.
			config := `{"typeChecking":{"enabled":false},"compiler":{"debug":true}}`
			if err := os.WriteFile(filepath.Join(dir, "flux.json"), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			compile := exec.Command(binary, "compile", name+".flux")
			compile.Dir = dir
			if output, err := compile.CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, output)
			}
			executable := filepath.Join(dir, "dist", name)
			if goruntime.GOOS == "windows" {
				executable += ".exe"
			}
			if err := os.Remove(filepath.Join(dir, name+".flux")); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			run := exec.Command(executable)
			run.Dir = t.TempDir()
			run.Stdout, run.Stderr = &stdout, &stderr
			err := run.Run()

			want := fixture.Outcome(fixtures.ModeDisabled)
			if expected := fixture.ExpectedStdout(fixtures.ModeDisabled); stdout.String() != expected {
				t.Errorf("stdout = %q, the fixture declares %q", stdout.String(), expected)
			}
			if want.Kind == conformance.KindOutput {
				if err != nil {
					t.Fatalf("the executable failed: %v\n%s", err, stderr.String())
				}
				return
			}
			if err == nil {
				t.Fatalf("the executable succeeded, the fixture declares %s", want)
			}
			// The executable resolved its position when it was built, so the
			// reported line and column are checked against the fixture's span
			// rather than recomputed from a source file it no longer needs.
			location := fmt.Sprintf("%s.flux:%d:%d: error[%s]",
				name, want.Span.StartLine, want.Span.StartColumn, want.Code)
			if !strings.Contains(stderr.String(), location) {
				t.Errorf("stderr does not report %q:\n%s", location, stderr.String())
			}
		})
	}
}

// representative picks the fixtures worth building an executable for: enough
// to exercise output, a runtime failure, and a call trace, without paying for a
// Go build per fixture.
func representative(suite []conformance.Fixture) []conformance.Fixture {
	wanted := map[string]bool{
		"EVL-PRINT":             true,
		"EVL-TOP-LEVEL-DISPLAY": true,
		"VAL-LIST-INDEX":        true,
		"VAL-DICT-MISSING":      true,
		"VAL-FN-ARITY":          true,
		"VAL-VOID":              true,
		"BND-CAPTURE":           true,
		"BND-SELF-RECURSION":    true,
		"VAL-FN-DEPTH":          true,
		"VAL-LOGICAL":           true,
		"VAL-INT-OVERFLOW":      true,
		"EVL-DISPLAY-OPAQUE":    true,
	}
	var selected []conformance.Fixture
	for _, fixture := range suite {
		if wanted[fixture.Rule] && fixture.Status == conformance.StatusImplemented {
			selected = append(selected, fixture)
		}
	}
	return selected
}

// testExists reports whether a test function of that name is defined in this
// package's test files.
func testExists(t *testing.T, name string) bool {
	t.Helper()
	matches, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	declaration := []byte("func " + name + "(t *testing.T)")
	for _, path := range matches {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(text, declaration) {
			return true
		}
	}
	return false
}
