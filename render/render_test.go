package render_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/render"
	"github.com/pranavms13/flux-lang/source"
)

// TestSnippetUnderlinesTheOffendingText pins the complete diagnostic headline,
// source snippet, notes, and help layout.
func TestSnippetUnderlinesTheOffendingText(t *testing.T) {
	const text = "let add = fn(a: int, b: int): int => a + b\nprint(add(\"5\", 10))\n"
	src := source.New(1, "main.flux", text)
	renderer := render.Renderer{Source: src}

	d := diagnostic.Error("T_ARGUMENT_TYPE", src.Span(strings.Index(text, `"5"`), strings.Index(text, `"5"`)+3),
		"expected int, found string").
		WithRelated(src.Span(13, 19), "parameter %q is declared as int here", "a")

	want := strings.Join([]string{
		`main.flux:2:11: error[T_ARGUMENT_TYPE]: expected int, found string`,
		`2 | print(add("5", 10))`,
		`  |           ^^^`,
		`  = note: parameter "a" is declared as int here at main.flux:1:14`,
	}, "\n")
	if got := renderer.Diagnostic(d); got != want {
		t.Errorf("rendered:\n%s\nwant:\n%s", got, want)
	}
}

// TestCaretAlignsWithExpandedTabs checks caret alignment using the same tab
// stops as source display columns.
func TestCaretAlignsWithExpandedTabs(t *testing.T) {
	// The line is indented with tabs. Printing it raw and counting columns
	// separately is how a caret ends up under the wrong character.
	const text = "let f = fn() =>\n\t\tbroken\n"
	src := source.New(1, "tabs.flux", text)
	renderer := render.Renderer{Source: src, TabWidth: 4}

	start := strings.Index(text, "broken")
	d := diagnostic.Error("B_UNDEFINED_VARIABLE", src.Span(start, start+len("broken")), "undefined variable: broken")

	lines := strings.Split(renderer.Diagnostic(d), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	code, underline := lines[1], lines[2]
	if !strings.HasPrefix(code, "2 |         broken") {
		t.Errorf("source line = %q, want the tabs expanded to eight columns", code)
	}
	if !strings.HasSuffix(underline, "^^^^^^") {
		t.Errorf("underline = %q, want six carets", underline)
	}
	if strings.Index(code, "broken") != strings.Index(underline, "^") {
		t.Errorf("the caret is not under the text:\n%s\n%s", code, underline)
	}
}

// TestNoColorByDefaultAndStyledWhenAsked verifies stable plain text and
// equivalent content when ANSI styling is enabled.
func TestNoColorByDefaultAndStyledWhenAsked(t *testing.T) {
	src := source.New(1, "color.flux", "let x = 1\n")
	d := diagnostic.Error("T_ANNOTATION_MISMATCH", src.Span(8, 9), "a mismatch")

	plain := render.Renderer{Source: src}.Diagnostic(d)
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("the default renderer emitted escape sequences: %q", plain)
	}

	styled := render.Renderer{Source: src, Color: true}.Diagnostic(d)
	if !strings.Contains(styled, "\x1b[") {
		t.Error("the styled renderer emitted no escape sequences")
	}
	// Styling must not change the text, only how it is drawn.
	if stripped := stripANSI(styled); stripped != plain {
		t.Errorf("styling changed the text:\n%s\nwant:\n%s", stripped, plain)
	}
}

// stripANSI removes styling escapes so tests can compare rendered content
// independently of color.
func stripANSI(text string) string {
	for {
		start := strings.Index(text, "\x1b[")
		if start < 0 {
			return text
		}
		end := strings.Index(text[start:], "m")
		if end < 0 {
			return text
		}
		text = text[:start] + text[start+end+1:]
	}
}

// TestColorEnabledRespectsNoColorAndNonTerminals checks that environment
// overrides and redirected output disable styling.
func TestColorEnabledRespectsNoColorAndNonTerminals(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if render.ColorEnabled(nil) {
		t.Error("NO_COLOR did not disable styling")
	}
	t.Setenv("NO_COLOR", "")
	// A buffer is not a terminal, so styling stays off without anyone asking.
	if render.ColorEnabled(new(strings.Builder)) {
		t.Error("styling was enabled for something that is not a terminal")
	}
}

// TestDiagnosticWithoutASourceHasNoSnippet checks useful source-less output
// without fabricated source text.
func TestDiagnosticWithoutASourceHasNoSnippet(t *testing.T) {
	d := diagnostic.Error("T_ANNOTATION_MISMATCH", source.Span{SourceID: 1, Start: 0, End: 4}, "a mismatch").
		WithNote("a note").
		WithHelp("try something else")

	got := render.Renderer{}.Diagnostic(d)
	want := "error[T_ANNOTATION_MISMATCH]: a mismatch\n  = note: a note\n  = help: try something else"
	if got != want {
		t.Errorf("rendered:\n%s\nwant:\n%s", got, want)
	}
}

// TestSpanFromAnotherSourceIsNotDrawn prevents a diagnostic span from being
// applied to an unrelated snapshot.
func TestSpanFromAnotherSourceIsNotDrawn(t *testing.T) {
	src := source.New(1, "a.flux", "let x = 1\n")
	d := diagnostic.Error("T_ANNOTATION_MISMATCH", source.Span{SourceID: 2, Start: 0, End: 3}, "elsewhere")

	got := render.Renderer{Source: src}.Diagnostic(d)
	if strings.Contains(got, "a.flux") || strings.Contains(got, "|") {
		t.Errorf("a span from another source was drawn against this one:\n%s", got)
	}
}

// TestMultilineSpanUnderlinesTheFirstLine verifies that underlining stops at
// the end of the displayed line.
func TestMultilineSpanUnderlinesTheFirstLine(t *testing.T) {
	const text = "let x = if true then {\n  1\n} else {\n  2\n}\n"
	src := source.New(1, "multiline.flux", text)
	d := diagnostic.Error("T_BRANCH_MISMATCH", src.Span(8, len(text)-1), "branches disagree")

	lines := strings.Split(render.Renderer{Source: src}.Diagnostic(d), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.HasPrefix(lines[1], "1 | let x = if true then {") {
		t.Errorf("source line = %q", lines[1])
	}
	// The span runs past the line, so the underline stops at its end rather
	// than running off into empty columns.
	carets := strings.Count(lines[2], "^")
	if carets != len("if true then {") {
		t.Errorf("underlined %d columns, want %d: %q", carets, len("if true then {"), lines[2])
	}
}

// TestFailureRendersTheCallTrace checks the fault headline, snippet, and
// ordered call-site frames.
func TestFailureRendersTheCallTrace(t *testing.T) {
	const text = "let inner = fn(x) => x + \"no\"\nlet outer = fn(y) => inner(y)\nprint(outer(1))\n"
	src := source.New(1, "trace.flux", text)

	failure := &fault.Error{
		Code:    fault.CodeOperandType,
		Message: "cannot apply + to int and string",
		Where:   source.Location{File: "trace.flux", Line: 1, Column: 22, Start: 21, End: 29},
		Trace: []fault.Frame{
			{Function: "inner", Call: source.Location{File: "trace.flux", Line: 2, Column: 22}},
			{Function: "outer", Call: source.Location{File: "trace.flux", Line: 3, Column: 7}},
		},
	}

	got := render.Renderer{Source: src}.Failure(failure)
	for _, want := range []string{
		"trace.flux:1:22: error[R_OPERAND_TYPE]: cannot apply + to int and string",
		`1 | let inner = fn(x) => x + "no"`,
		"  in inner, called at trace.flux:2:22",
		"  in outer, called at trace.flux:3:7",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered output does not contain %q:\n%s", want, got)
		}
	}
}

// TestFailureSnippetRequiresMatchingSource keeps a failure's headline and
// snippet tied to the same file, including when source text is unavailable.
func TestFailureSnippetRequiresMatchingSource(t *testing.T) {
	src := source.New(7, "failure.flux", "missing")
	failure := fault.UndefinedValue("missing").At(src.Locate(src.Whole()))
	for _, test := range []struct {
		name    string
		src     *source.Source
		snippet bool
	}{
		{"matching", src, true},
		{"different file", source.New(7, "other.flux", "unrelated"), false},
		{"no source", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := render.Renderer{Source: test.src}.Failure(failure)
			if !strings.HasPrefix(got, "failure.flux:1:1: error[R_UNDEFINED_VALUE]") {
				t.Errorf("failure lost its location: %q", got)
			}
			if strings.Contains(got, "1 | ") != test.snippet {
				t.Errorf("snippet present = %t, want %t: %q", strings.Contains(got, "1 | "), test.snippet, got)
			}
		})
	}
}

// TestInternalFailureSaysItIsAFluxBug verifies that internal failures are
// identified as implementation defects.
func TestInternalFailureSaysItIsAFluxBug(t *testing.T) {
	failure := &fault.Error{Code: diagnostic.CodeInternal, Message: "unknown opcode 99"}
	got := render.Renderer{}.Failure(failure)

	if !strings.HasPrefix(got, "internal error[X_INTERNAL]: unknown opcode 99") {
		t.Errorf("rendered: %q", got)
	}
	if !strings.Contains(got, "this is a bug in Flux") {
		t.Errorf("an internal failure does not say whose bug it is: %q", got)
	}
}

// TestToolErrorLooksUnlikeALanguageError checks that tool failures have no
// language code or source caret.
func TestToolErrorLooksUnlikeALanguageError(t *testing.T) {
	got := render.Renderer{}.ToolError(diagnostic.Tool("read source file", "missing.flux", errNotExist{}))

	if strings.Contains(got, "[") {
		t.Errorf("a tool error was given a diagnostic code: %q", got)
	}
	if !strings.HasPrefix(got, "error: read source file missing.flux") {
		t.Errorf("rendered: %q", got)
	}
}

type errNotExist struct{}

// Error supplies a deterministic missing-file message for renderer tests.
func (errNotExist) Error() string { return "no such file or directory" }

// TestJSONIsVersionedAndCarriesBothUnits checks the schema version, byte
// offsets, and display columns for structured diagnostics.
func TestJSONIsVersionedAndCarriesBothUnits(t *testing.T) {
	const text = "let x: int = \"hello\"\n"
	src := source.New(1, "json.flux", text)
	d := diagnostic.Error("T_ANNOTATION_MISMATCH", src.Span(13, 20), "expected int, found string").
		WithRelated(src.Span(5, 10), "declared here").
		WithNote("a note").
		WithHelp("some help")

	encoded, err := render.Renderer{Source: src}.JSON([]diagnostic.Diagnostic{d})
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Version     int `json:"version"`
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Primary  struct {
				File   string `json:"file"`
				Line   int    `json:"line"`
				Column int    `json:"column"`
				Start  int    `json:"start"`
				End    int    `json:"end"`
			} `json:"primary"`
			Related []struct {
				Message  string `json:"message"`
				Location struct {
					Line int `json:"line"`
				} `json:"location"`
			} `json:"related"`
			Notes []string `json:"notes"`
			Help  string   `json:"help"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != render.JSONVersion {
		t.Errorf("version = %d, want %d", document.Version, render.JSONVersion)
	}
	if len(document.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(document.Diagnostics))
	}
	entry := document.Diagnostics[0]
	if entry.Code != "T_ANNOTATION_MISMATCH" || entry.Severity != "error" {
		t.Errorf("code %q severity %q", entry.Code, entry.Severity)
	}
	// Both units are reported because neither can be derived from the other
	// without the source text.
	if entry.Primary.File != "json.flux" || entry.Primary.Line != 1 || entry.Primary.Column != 14 {
		t.Errorf("primary position = %+v", entry.Primary)
	}
	if entry.Primary.Start != 13 || entry.Primary.End != 20 {
		t.Errorf("primary byte range = %d..%d, want 13..20", entry.Primary.Start, entry.Primary.End)
	}
	if len(entry.Related) != 1 || entry.Related[0].Message != "declared here" {
		t.Errorf("related = %+v", entry.Related)
	}
	if len(entry.Notes) != 1 || entry.Help != "some help" {
		t.Errorf("notes = %v, help = %q", entry.Notes, entry.Help)
	}
}

// TestJSONOmitsWhatIsAbsent checks that unavailable locations and optional
// details are omitted from JSON.
func TestJSONOmitsWhatIsAbsent(t *testing.T) {
	encoded, err := render.Renderer{}.JSON([]diagnostic.Diagnostic{
		diagnostic.Error("X_INTERNAL", source.NoSpan, "no location"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"primary", "related", "notes", "help", "trace"} {
		if strings.Contains(string(encoded), `"`+absent+`"`) {
			t.Errorf("the document contains an empty %q field:\n%s", absent, encoded)
		}
	}
}

// TestJSONFailureUsesTheSameDocumentShape verifies that runtime failures use
// the versioned diagnostic schema and retain call traces.
func TestJSONFailureUsesTheSameDocumentShape(t *testing.T) {
	encoded, err := render.Renderer{}.JSONFailure(&fault.Error{
		Code:    fault.CodeMissingKey,
		Message: `dictionary has no key "missing"`,
		Where:   source.Location{File: "a.flux", Line: 3, Column: 5, Start: 20, End: 31},
		Trace:   []fault.Frame{{Function: "lookup", Call: source.Location{File: "a.flux", Line: 7, Column: 2}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Version     int `json:"version"`
		Diagnostics []struct {
			Code  string `json:"code"`
			Trace []struct {
				Function string `json:"function"`
				Call     struct {
					Line int `json:"line"`
				} `json:"call"`
			} `json:"trace"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != render.JSONVersion {
		t.Errorf("version = %d, want %d", document.Version, render.JSONVersion)
	}
	if len(document.Diagnostics) != 1 || document.Diagnostics[0].Code != "R_MISSING_KEY" {
		t.Fatalf("diagnostics = %+v", document.Diagnostics)
	}
	trace := document.Diagnostics[0].Trace
	if len(trace) != 1 || trace[0].Function != "lookup" || trace[0].Call.Line != 7 {
		t.Errorf("trace = %+v", trace)
	}
}

// TestSeveralDiagnosticsRenderInOrder checks that text rendering preserves the
// supplied diagnostic order.
func TestSeveralDiagnosticsRenderInOrder(t *testing.T) {
	src := source.New(1, "many.flux", "let a = 1\nlet b = 2\n")
	got := render.Renderer{Source: src}.Diagnostics([]diagnostic.Diagnostic{
		diagnostic.Error("T_A", src.Span(0, 3), "first"),
		diagnostic.Warning("T_B", src.Span(10, 13), "second"),
	})
	first, second := strings.Index(got, "first"), strings.Index(got, "second")
	if first < 0 || second < 0 || first > second {
		t.Errorf("rendered:\n%s", got)
	}
	if !strings.Contains(got, "warning[T_B]") {
		t.Errorf("the warning was not rendered as one:\n%s", got)
	}
}

// TestTextAndJSONDescribeTheSameDiagnostic is a Phase 1 completion criterion:
// the two outputs are two renderings of one record, so a consumer of either
// must be able to reach the same conclusion. Each is produced from the record
// independently, which is exactly how they drift apart.
func TestTextAndJSONDescribeTheSameDiagnostic(t *testing.T) {
	const program = "let add = fn(a: int, b: int): int => a + b\nprint(add(\"5\", 10))\n"
	src := source.New(1, "same.flux", program)
	renderer := render.Renderer{Source: src}

	d := diagnostic.Error("T_ARGUMENT_TYPE",
		src.Span(strings.Index(program, `"5"`), strings.Index(program, `"5"`)+3),
		"argument 1 has type string, expected int").
		WithRelated(src.Span(13, 19), "parameter %q is declared as int here", "a").
		WithNote("a note").
		WithHelp("some help")

	text := renderer.Diagnostic(d)
	encoded, err := renderer.JSON([]diagnostic.Diagnostic{d})
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Primary  struct {
				File   string `json:"file"`
				Line   int    `json:"line"`
				Column int    `json:"column"`
			} `json:"primary"`
			Related []struct {
				Message  string `json:"message"`
				Location struct {
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"location"`
			} `json:"related"`
			Notes []string `json:"notes"`
			Help  string   `json:"help"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics in JSON, want 1", len(document.Diagnostics))
	}
	entry := document.Diagnostics[0]

	// Everything the JSON asserts must be findable in the text, in the form the
	// text uses. The reverse does not hold: the text also draws a snippet,
	// which JSON leaves to the consumer that has the source.
	for _, want := range []string{
		entry.Code,
		entry.Severity,
		entry.Message,
		fmt.Sprintf("%s:%d:%d", entry.Primary.File, entry.Primary.Line, entry.Primary.Column),
		entry.Help,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the text rendering omits %q that the JSON reports:\n%s", want, text)
		}
	}
	for _, note := range entry.Notes {
		if !strings.Contains(text, note) {
			t.Errorf("the text rendering omits the note %q", note)
		}
	}
	for _, related := range entry.Related {
		if !strings.Contains(text, related.Message) {
			t.Errorf("the text rendering omits the related message %q", related.Message)
		}
		where := fmt.Sprintf("%d:%d", related.Location.Line, related.Location.Column)
		if !strings.Contains(text, where) {
			t.Errorf("the text rendering omits the related position %q", where)
		}
	}
}
