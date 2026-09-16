package diagnostic_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/source"
)

func spanAt(start, end int) source.Span {
	return source.Span{SourceID: 1, Start: start, End: end}
}

func TestErrorAndWarningCarryCodeSeverityAndSpan(t *testing.T) {
	span := spanAt(4, 7)

	err := diagnostic.Error("T_ARGUMENT_TYPE", span, "expected %s, found %s", "int", "string")
	if err.Code != "T_ARGUMENT_TYPE" || err.Severity != diagnostic.SeverityError {
		t.Errorf("Error() produced %+v, want an error with code T_ARGUMENT_TYPE", err)
	}
	if got, want := err.Message, "expected int, found string"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
	if err.Primary != span || !err.HasLocation() {
		t.Errorf("Primary = %+v, want %+v with a location", err.Primary, span)
	}

	warning := diagnostic.Warning("T_ARGUMENT_TYPE", span, "expected int")
	if warning.Severity != diagnostic.SeverityWarning {
		t.Errorf("Warning severity = %v, want warning", warning.Severity)
	}
	if !diagnostic.Error("X_INTERNAL", source.NoSpan, "boom").HasLocation() {
		return
	}
	t.Error("a diagnostic built on NoSpan reports that it has a location")
}

func TestBuildersDoNotShareStateBetweenBranches(t *testing.T) {
	base := diagnostic.Error("T_ARGUMENT_TYPE", spanAt(0, 1), "mismatch").
		WithNote("shared note")

	first := base.WithNote("first").WithRelated(spanAt(2, 3), "declared here")
	second := base.WithNote("second")

	if got, want := len(base.Notes), 1; got != want {
		t.Errorf("the base diagnostic gained notes: %d, want %d", got, want)
	}
	if got, want := first.Notes[1], "first"; got != want {
		t.Errorf("first.Notes[1] = %q, want %q", got, want)
	}
	if got, want := second.Notes[1], "second"; got != want {
		t.Errorf("second.Notes[1] = %q, want %q (branches must not share an array)", got, want)
	}
	if len(second.Related) != 0 {
		t.Errorf("second gained related labels from first: %+v", second.Related)
	}
	if got, want := first.Related[0].Message, "declared here"; got != want {
		t.Errorf("related message = %q, want %q", got, want)
	}
}

func TestWithSeverityPreservesCodeAndLocation(t *testing.T) {
	original := diagnostic.Error("T_ARGUMENT_TYPE", spanAt(4, 7), "expected int")
	downgraded := original.WithSeverity(diagnostic.SeverityWarning)

	if downgraded.Severity != diagnostic.SeverityWarning {
		t.Errorf("severity = %v, want warning", downgraded.Severity)
	}
	if downgraded.Code != original.Code || downgraded.Primary != original.Primary ||
		downgraded.Message != original.Message {
		t.Errorf("downgrading changed more than severity: %+v", downgraded)
	}
	if original.Severity != diagnostic.SeverityError {
		t.Error("downgrading mutated the original diagnostic")
	}
}

func TestWithHelpReplacesRatherThanAccumulates(t *testing.T) {
	d := diagnostic.Error("B_UNDEFINED", spanAt(0, 3), "undefined variable").
		WithHelp("declare %s first", "x").
		WithHelp("did you mean %s?", "y")
	if got, want := d.Help, "did you mean y?"; got != want {
		t.Errorf("Help = %q, want %q", got, want)
	}
}

func TestSeverityNames(t *testing.T) {
	for _, test := range []struct {
		severity diagnostic.Severity
		want     string
	}{
		{diagnostic.SeverityError, "error"},
		{diagnostic.SeverityWarning, "warning"},
		{diagnostic.SeverityInfo, "info"},
		{diagnostic.Severity(42), "severity(42)"},
	} {
		if got := test.severity.String(); got != test.want {
			t.Errorf("Severity(%d).String() = %q, want %q", int(test.severity), got, test.want)
		}
	}
}

func TestCodeGroups(t *testing.T) {
	for _, test := range []struct {
		code  diagnostic.Code
		group diagnostic.Group
		ok    bool
	}{
		{"S_UNEXPECTED_TOKEN", diagnostic.GroupSyntax, true},
		{"B_UNDEFINED_VARIABLE", diagnostic.GroupBinding, true},
		{"T_ARGUMENT_TYPE", diagnostic.GroupType, true},
		{"R_INDEX_OUT_OF_BOUNDS", diagnostic.GroupRuntime, true},
		{"X_INTERNAL", diagnostic.GroupInternal, true},
		{"Q_UNKNOWN", "", false},
		{"NOPREFIX", "", false},
		{"", "", false},
	} {
		group, ok := test.code.Group()
		if group != test.group || ok != test.ok {
			t.Errorf("Code(%q).Group() = (%q, %v), want (%q, %v)", test.code, group, ok, test.group, test.ok)
		}
	}
}

func TestRegisterRejectsUnknownGroupsAndDuplicates(t *testing.T) {
	code := diagnostic.Register("T_TEST_ONLY_CODE", "a code declared by this test")
	description, ok := diagnostic.Describe(code)
	if !ok || description != "a code declared by this test" {
		t.Errorf("Describe(%q) = (%q, %v), want the registered description", code, description, ok)
	}
	if _, ok := diagnostic.Describe("T_NEVER_REGISTERED"); ok {
		t.Error("Describe reported a description for an unregistered code")
	}

	assertPanics(t, "duplicate code", func() {
		diagnostic.Register("T_TEST_ONLY_CODE", "declared twice")
	})
	assertPanics(t, "code without a reserved group", func() {
		diagnostic.Register("Q_NO_SUCH_GROUP", "invalid")
	})

	var found bool
	for _, registered := range diagnostic.Registered() {
		if registered == code {
			found = true
		}
	}
	if !found {
		t.Errorf("Registered() omits %q", code)
	}
	if !sortedAscending(diagnostic.Registered()) {
		t.Error("Registered() is not sorted")
	}
}

func sortedAscending(codes []diagnostic.Code) bool {
	for i := 1; i < len(codes); i++ {
		if codes[i-1] >= codes[i] {
			return false
		}
	}
	return true
}

func assertPanics(t *testing.T, what string, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	call()
}

func TestInternalDiagnosticIsDistinguishableFromLanguageErrors(t *testing.T) {
	d := diagnostic.Internal(source.NoSpan, "unreachable opcode %d", 99)

	group, ok := d.Code.Group()
	if !ok || group != diagnostic.GroupInternal {
		t.Errorf("Internal() produced code %q in group %q, want the internal group", d.Code, group)
	}
	if len(d.Notes) == 0 {
		t.Error("an internal diagnostic must say it is a Flux bug")
	}
	if got, want := d.Message, "unreachable opcode 99"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
}

func TestToolErrorIsSeparateFromLanguageErrors(t *testing.T) {
	err := diagnostic.Tool("read source file", "missing.flux", fs.ErrNotExist)

	if got, want := err.Error(), "read source file missing.flux: file does not exist"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Error("errors.Is cannot see the wrapped cause")
	}

	var tool *diagnostic.ToolError
	if !errors.As(fmt.Errorf("compile: %w", err), &tool) {
		t.Error("errors.As cannot recover a wrapped ToolError")
	}
	if got, want := (&diagnostic.ToolError{Op: "locate Go toolchain"}).Error(), "locate Go toolchain"; got != want {
		t.Errorf("Error() with no path or cause = %q, want %q", got, want)
	}
	if got, want := (&diagnostic.ToolError{Op: "read", Path: "a.flux"}).Error(), "read a.flux"; got != want {
		t.Errorf("Error() with no cause = %q, want %q", got, want)
	}
	if got, want := (&diagnostic.ToolError{Op: "decode", Err: fs.ErrInvalid}).Error(), "decode: invalid argument"; got != want {
		t.Errorf("Error() with no path = %q, want %q", got, want)
	}
}

func TestBagOrdersByPositionThenCodeThenArrival(t *testing.T) {
	var bag diagnostic.Bag
	bag.Add(diagnostic.Error("T_LATER", spanAt(30, 33), "third"))
	bag.Add(diagnostic.Error("T_SECOND", spanAt(10, 12), "second by code"))
	bag.Add(diagnostic.Error("T_FIRST", spanAt(10, 12), "first by code"))
	bag.Add(diagnostic.Error("S_SYNTAX", spanAt(4, 8), "earliest position"))
	bag.Add(diagnostic.Error("T_FIRST", spanAt(10, 12), "same code, later arrival"))
	bag.Add(diagnostic.Error("T_SHORTER", spanAt(10, 11), "shorter span at the same start"))

	want := []string{
		"earliest position",
		"shorter span at the same start",
		"first by code",
		"same code, later arrival",
		"second by code",
		"third",
	}
	assertMessages(t, bag.All(), want)

	// Ordering must not depend on the order diagnostics happened to arrive.
	var shuffled diagnostic.Bag
	shuffled.Add(diagnostic.Error("T_SHORTER", spanAt(10, 11), "shorter span at the same start"))
	shuffled.Add(diagnostic.Error("T_FIRST", spanAt(10, 12), "first by code"))
	shuffled.Add(diagnostic.Error("S_SYNTAX", spanAt(4, 8), "earliest position"))
	shuffled.Add(diagnostic.Error("T_FIRST", spanAt(10, 12), "same code, later arrival"))
	shuffled.Add(diagnostic.Error("T_LATER", spanAt(30, 33), "third"))
	shuffled.Add(diagnostic.Error("T_SECOND", spanAt(10, 12), "second by code"))
	assertMessages(t, shuffled.All(), want)
}

func TestBagOrdersAcrossSources(t *testing.T) {
	var bag diagnostic.Bag
	bag.Add(diagnostic.Error("T_A", source.Span{SourceID: 2, Start: 0, End: 1}, "second file"))
	bag.Add(diagnostic.Error("T_A", source.Span{SourceID: 1, Start: 99, End: 100}, "first file"))
	assertMessages(t, bag.All(), []string{"first file", "second file"})
}

func TestBagDeduplicatesIdenticalReports(t *testing.T) {
	var bag diagnostic.Bag
	first := bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(4, 5), "undefined variable x"))
	again := bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(4, 5), "undefined variable x"))

	if first != again {
		t.Errorf("re-reporting returned ref %d, want the existing ref %d", again, first)
	}
	assertMessages(t, bag.All(), []string{"undefined variable x"})

	// Anything that differs is a different diagnostic and must survive.
	bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(4, 5), "undefined variable y"))
	bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(9, 10), "undefined variable x"))
	bag.Add(diagnostic.Error("B_OTHER", spanAt(4, 5), "undefined variable x"))
	bag.Add(diagnostic.Warning("B_UNDEFINED", spanAt(4, 5), "undefined variable x"))
	if got, want := bag.Len(), 5; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

func TestBagSuppressesConsequencesOfOneRootFailure(t *testing.T) {
	var bag diagnostic.Bag
	root := bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(8, 9), "undefined variable x"))
	call := bag.AddCausedBy(diagnostic.Error("T_NOT_CALLABLE", spanAt(8, 12), "cannot call unknown value"), root)
	bag.AddCausedBy(diagnostic.Error("T_LET_TYPE", spanAt(0, 12), "cannot infer type of a"), call)
	bag.Add(diagnostic.Error("T_UNRELATED", spanAt(20, 24), "unrelated failure"))

	assertMessages(t, bag.All(), []string{"undefined variable x", "unrelated failure"})
	// The full list is sorted by position like any other, so the enclosing
	// declaration at offset 0 comes first regardless of when it was recorded.
	assertMessages(t, bag.AllIncludingSuppressed(), []string{
		"cannot infer type of a",
		"undefined variable x",
		"cannot call unknown value",
		"unrelated failure",
	})
	if got, want := bag.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d (suppressed diagnostics are not reported)", got, want)
	}
}

func TestBagSeverityQueries(t *testing.T) {
	var bag diagnostic.Bag
	if !bag.Empty() || bag.HasErrors() || bag.HasWarnings() {
		t.Error("a fresh bag reports content")
	}

	bag.Add(diagnostic.Warning("T_MODE", spanAt(0, 1), "downgraded mismatch"))
	if bag.HasErrors() {
		t.Error("HasErrors() is true with only a warning")
	}
	if !bag.HasWarnings() || bag.Empty() {
		t.Error("HasWarnings() is false after adding a warning")
	}

	root := bag.Add(diagnostic.Error("B_UNDEFINED", spanAt(4, 5), "undefined variable"))
	bag.AddCausedBy(diagnostic.Error("T_CASCADE", spanAt(4, 9), "consequence"), root)
	if !bag.HasErrors() {
		t.Error("HasErrors() is false after adding an error")
	}

	assertMessages(t, bag.WithSeverity(diagnostic.SeverityError), []string{"undefined variable"})
	assertMessages(t, bag.WithSeverity(diagnostic.SeverityWarning), []string{"downgraded mismatch"})
	if got := bag.WithSeverity(diagnostic.SeverityInfo); len(got) != 0 {
		t.Errorf("WithSeverity(info) = %+v, want none", got)
	}
}

func TestBagSuppressedErrorAloneDoesNotFailACommand(t *testing.T) {
	var bag diagnostic.Bag
	root := bag.Add(diagnostic.Warning("T_MODE", spanAt(0, 4), "downgraded"))
	bag.AddCausedBy(diagnostic.Error("T_CASCADE", spanAt(0, 8), "consequence of a downgraded error"), root)

	if bag.HasErrors() {
		t.Error("HasErrors() is true for a suppressed error; the command would fail on a hidden diagnostic")
	}
}

func TestBagExtend(t *testing.T) {
	var bag diagnostic.Bag
	bag.Extend([]diagnostic.Diagnostic{
		diagnostic.Error("T_B", spanAt(10, 12), "second"),
		diagnostic.Error("T_A", spanAt(2, 4), "first"),
		diagnostic.Error("T_A", spanAt(2, 4), "first"),
	})
	assertMessages(t, bag.All(), []string{"first", "second"})
}

func assertMessages(t *testing.T, got []diagnostic.Diagnostic, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d diagnostics, want %d: %s", len(got), len(want), messages(got))
	}
	for i := range want {
		if got[i].Message != want[i] {
			t.Errorf("diagnostic %d = %q, want %q (full order: %s)", i, got[i].Message, want[i], messages(got))
		}
	}
}

func messages(diagnostics []diagnostic.Diagnostic) string {
	collected := make([]string, len(diagnostics))
	for i, d := range diagnostics {
		collected[i] = d.Message
	}
	return fmt.Sprint(collected)
}
