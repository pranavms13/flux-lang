package conformance

import (
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/internal/fixtures"
)

// header is the smallest well-formed implemented fixture, used as the base for
// tests that change one thing about it.
const header = `//! rule: VAL-ADD-INT
//! about: + adds two ints
//! status: implemented
//! all: output
//! stdout: "5\n"
print(2 + 3)
`

func TestParseReadsAnImplementedFixture(t *testing.T) {
	fixture, err := Parse("values/add.flux", header)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.Rule != "VAL-ADD-INT" || fixture.Status != StatusImplemented {
		t.Errorf("rule %q status %q", fixture.Rule, fixture.Status)
	}
	if fixture.Stdout != "5\n" {
		t.Errorf("stdout = %q, want %q", fixture.Stdout, "5\n")
	}
	for _, mode := range fixtures.Modes {
		if got := fixture.Outcome(mode); got.Kind != KindOutput {
			t.Errorf("%s mode is %q, want %q", mode, got.Kind, KindOutput)
		}
	}
	// The whole file is kept, header included, because the declared spans are
	// offsets into what the parser will actually read.
	if fixture.Source != header {
		t.Error("the fixture did not keep its own source")
	}
}

func TestParseOverridesAllWithASpecificMode(t *testing.T) {
	text := `//! rule: R
//! about: a
//! status: implemented
//! all: static-error T_X at 1:1..1:2
//! disabled: runtime-error R_X at 3:4..3:5
x
`
	fixture, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	if got := fixture.Outcome(fixtures.ModeStrict); got.Kind != KindStaticError || got.Code != "T_X" {
		t.Errorf("strict = %s, want a static T_X", got)
	}
	got := fixture.Outcome(fixtures.ModeDisabled)
	want := Outcome{Kind: KindRuntimeError, Code: "R_X",
		Span: Span{StartLine: 3, StartColumn: 4, EndLine: 3, EndColumn: 5}}
	if got != want {
		t.Errorf("disabled = %s, want %s", got, want)
	}
}

func TestParseStopsAtTheFirstNonDirective(t *testing.T) {
	// A directive further down the file must not change what the fixture
	// asserts: a reader checks the header, not every comment in the program.
	text := header + "//! stdout: \"ignored\"\n"
	fixture, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.Stdout != "5\n" {
		t.Errorf("a later directive changed stdout to %q", fixture.Stdout)
	}
}

func TestParseReadsWarnings(t *testing.T) {
	text := `//! rule: R
//! about: a
//! status: implemented
//! all: output
//! lenient-warning: T_CONDITION_TYPE at 6:10..6:11
//! stdout: "x\n"
print("x")
`
	fixture, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	want := Warning{Code: "T_CONDITION_TYPE",
		Span: Span{StartLine: 6, StartColumn: 10, EndLine: 6, EndColumn: 11}}
	if got := fixture.Warnings[fixtures.ModeLenient]; got != want {
		t.Errorf("lenient warning = %s, want %s", got, want)
	}
	if _, declared := fixture.Warnings[fixtures.ModeStrict]; declared {
		t.Error("a mode that declared no warning has one")
	}
}

func TestParseRejectsIncoherentFixtures(t *testing.T) {
	cases := []struct {
		name, text, wants string
	}{
		{
			name:  "no rule",
			text:  "//! about: a\n//! status: implemented\n//! all: output\n//! stdout: \"\"\n",
			wants: "must name the docs/SPEC.md rule",
		},
		{
			name:  "missing mode",
			text:  "//! rule: R\n//! about: a\n//! status: implemented\n//! strict: output\n//! stdout: \"\"\n",
			wants: "no outcome declared for mode",
		},
		{
			name:  "output without stdout",
			text:  "//! rule: R\n//! about: a\n//! status: implemented\n//! all: output\n",
			wants: "must declare stdout",
		},
		{
			name:  "error without a span",
			text:  "//! rule: R\n//! about: a\n//! status: implemented\n//! all: static-error T_X\n",
			wants: "line:col..line:col",
		},
		{
			name: "planned without a milestone",
			text: "//! rule: R\n//! about: a\n//! status: planned\n//! specified: output\n" +
				"//! specified-stdout: \"x\\n\"\n//! all: output\n//! stdout: \"y\\n\"\n",
			wants: "must name the phase",
		},
		{
			name: "planned that is already satisfied",
			text: "//! rule: R\n//! about: a\n//! status: planned\n//! milestone: phase-3\n" +
				"//! specified: output\n//! specified-stdout: \"x\\n\"\n//! all: output\n//! stdout: \"x\\n\"\n",
			wants: "already matches the specified outcome",
		},
		{
			name: "implemented with a specification",
			text: "//! rule: R\n//! about: a\n//! status: implemented\n//! specified: output\n" +
				"//! all: output\n//! stdout: \"x\\n\"\n",
			wants: "outcomes are the specification",
		},
		{
			name:  "unknown directive",
			text:  "//! rule: R\n//! nonsense: a\n",
			wants: "unknown directive",
		},
		{
			name:  "unknown mode warning",
			text:  "//! rule: R\n//! loose-warning: T_X at 1:1..1:2\n",
			wants: "unknown mode",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse("f.flux", test.text)
			if err == nil {
				t.Fatalf("accepted a fixture that %s", test.wants)
			}
			if !strings.Contains(err.Error(), test.wants) {
				t.Errorf("error is %q, want it to mention %q", err, test.wants)
			}
		})
	}
}

func TestSatisfiesSpecificationNeedsEveryMode(t *testing.T) {
	// A planned rule such as "in every mode" is often already met by the
	// strictest one. That is not the rule being implemented, so the fixture
	// stays planned until every mode agrees.
	text := `//! rule: R
//! about: a
//! status: planned
//! milestone: phase-3
//! specified: static-error T_X at 1:1..1:2
//! strict: static-error T_X at 1:1..1:2
//! lenient: output
//! warn-only: output
//! disabled: output
//! stdout: "x\n"
print("x")
`
	fixture, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.SatisfiesSpecification() {
		t.Error("a rule met in one mode of four counts as satisfied")
	}
}

func TestCodesCollectsOnlyAssertedDiagnostics(t *testing.T) {
	planned, err := Parse("f.flux", `//! rule: R
//! about: a
//! status: planned
//! milestone: phase-3
//! specified: static-error T_FUTURE at 1:1..1:2
//! all: runtime-error R_TODAY at 1:1..1:2
//! lenient-warning: T_WARNING at 1:1..1:2
x
`)
	if err != nil {
		t.Fatal(err)
	}
	codes := Codes([]Fixture{planned})
	if len(codes) != 2 || codes[0] != "R_TODAY" || codes[1] != "T_WARNING" {
		t.Errorf("codes = %v, want the current error and warning, without the future code", codes)
	}
}

func TestPlannedOutcomesVaryByMode(t *testing.T) {
	text := `//! rule: TYP-INFERENCE-CONSTRAINTS
//! about: inference rejects a bad call only when checking can stop execution
//! status: planned
//! milestone: phase-4
//! specified: runtime-error R_OPERAND_TYPE at 1:1..1:2
//! specified-strict: static-error T_ARGUMENT_TYPE at 2:1..2:2
//! specified-lenient: static-error T_ARGUMENT_TYPE at 2:1..2:2
//! all: runtime-error R_OPERAND_TYPE at 1:1..1:2
`
	f, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	if f.Specified[fixtures.ModeDisabled].Kind != KindRuntimeError ||
		f.Specified[fixtures.ModeWarnOnly].Kind != KindRuntimeError ||
		f.Specified[fixtures.ModeStrict].Kind != KindStaticError {
		t.Fatalf("specified overrides lost mode differences: %v", f.Specified)
	}
	f.Outcomes[fixtures.ModeStrict] = f.Specified[fixtures.ModeStrict]
	if f.SatisfiesSpecification() {
		t.Fatal("lenient checking still lacks inference")
	}
	f.Outcomes[fixtures.ModeLenient] = f.Specified[fixtures.ModeLenient]
	if !f.SatisfiesSpecification() {
		t.Fatal("a completed rule with different per-mode outcomes was not recognized")
	}
	if err := f.Validate(); err == nil || !strings.Contains(err.Error(), "mark the rule implemented") {
		t.Fatalf("completed rule was not rejected as planned: %v", err)
	}

	// Every future mode still needs an expectation when the shorthand is omitted.
	missing := strings.Replace(text, "//! specified: runtime-error R_OPERAND_TYPE at 1:1..1:2\n", "", 1)
	if _, err := Parse("f.flux", missing); err == nil || !strings.Contains(err.Error(), "specified-warn-only") {
		t.Fatalf("missing future mode was not rejected: %v", err)
	}
	unknown := strings.Replace(text, "specified-strict:", "specified-loose:", 1)
	if _, err := Parse("f.flux", unknown); err == nil || !strings.Contains(err.Error(), "unknown mode") {
		t.Fatalf("unknown future mode was not rejected: %v", err)
	}
}

func TestPartialOutputAndStaticRejection(t *testing.T) {
	text := `//! rule: VAL-VOID
//! about: print takes effect before a runtime failure, but not before static rejection
//! status: implemented
//! all: static-error T_OPERAND_TYPE at 1:1..1:2
//! warn-only: runtime-error R_OPERAND_TYPE at 1:1..1:2
//! disabled: runtime-error R_OPERAND_TYPE at 1:1..1:2
//! stdout: "before\n"
`
	f, err := Parse("f.flux", text)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range fixtures.Modes {
		want := "before\n"
		if mode == fixtures.ModeStrict || mode == fixtures.ModeLenient {
			want = ""
		}
		if got := f.ExpectedStdout(mode); got != want {
			t.Errorf("%s stdout = %q, want %q", mode, got, want)
		}
	}
	withoutOutput := strings.Replace(text, "//! stdout: \"before\\n\"\n", "", 1)
	f, err = Parse("f.flux", withoutOutput)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.ExpectedStdout(fixtures.ModeDisabled); got != "" {
		t.Errorf("omitted error output = %q, want empty", got)
	}
}

func TestPlannedRuntimeFailuresComparePartialOutput(t *testing.T) {
	f, err := Parse("f.flux", `//! rule: R
//! about: the rule changes the output before a failure
//! status: planned
//! milestone: phase-3
//! specified: runtime-error R_X at 1:1..1:2
//! specified-stdout: "new\n"
//! all: runtime-error R_X at 1:1..1:2
//! stdout: "old\n"
`)
	if err != nil {
		t.Fatal(err)
	}
	if f.SatisfiesSpecification() {
		t.Fatal("matching failures with different partial output counted as implemented")
	}
	f.Stdout = "new\n"
	if !f.SatisfiesSpecification() {
		t.Fatal("matching failures and partial output did not count as implemented")
	}
}

func TestParseSpanRejectsMalformedPositions(t *testing.T) {
	for _, text := range []string{"1:1", "1:1..1", "0:1..1:2", "a:1..1:2", "1:0..1:2"} {
		if _, err := ParseSpan(text); err == nil {
			t.Errorf("accepted span %q", text)
		}
	}
	span, err := ParseSpan("3:4..5:6")
	if err != nil {
		t.Fatal(err)
	}
	if span.String() != "3:4..5:6" {
		t.Errorf("span round-trips as %q", span.String())
	}
}
