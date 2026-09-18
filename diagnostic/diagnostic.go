// Package diagnostic describes what a Flux compilation reported, separately
// from how any consumer displays it.
//
// A diagnostic is data: a stable code, a severity, a message, a primary span,
// optional related locations, notes, and help. It carries no terminal escapes,
// no line snippets, and no JSON tags-for-display, because the same record has
// to render as a terminal message, as a JSON object, and as an LSP diagnostic
// without being produced three times.
//
// Like [github.com/pranavms13/flux-lang/source], this package depends on
// nothing else in the module so the standalone executable bundle can include it
// alongside the VM.
package diagnostic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pranavms13/flux-lang/source"
)

// Severity says how a diagnostic affects the outcome of a command. It is
// deliberately independent of the code: the strict and warn-only configuration
// modes adjust severity in one place while leaving codes and spans untouched.
type Severity int

const (
	// SeverityError fails the command that produced it.
	SeverityError Severity = iota
	// SeverityWarning describes a problem that does not fail the command.
	SeverityWarning
	// SeverityInfo describes something worth surfacing in an editor but not
	// worth interrupting a build for.
	SeverityInfo
)

// String returns the lowercase name used in rendered output and JSON.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// Group is the reserved prefix of a diagnostic code. The group tells a reader
// which stage found the problem before they look the code up.
type Group string

const (
	// GroupSyntax covers lexing and parsing failures.
	GroupSyntax Group = "S"
	// GroupBinding covers name resolution: undefined, duplicate, and
	// out-of-scope identifiers.
	GroupBinding Group = "B"
	// GroupType covers the type checker.
	GroupType Group = "T"
	// GroupRuntime covers failures raised while a program executes, by either
	// the interpreter or the VM.
	GroupRuntime Group = "R"
	// GroupInternal covers defects in Flux itself. A user program must never
	// be able to provoke one; if it does, that is the bug to fix, and the
	// distinct group keeps it from being mistaken for a language error.
	GroupInternal Group = "X"
)

// Groups lists every reserved group in reporting order.
var Groups = []Group{GroupSyntax, GroupBinding, GroupType, GroupRuntime, GroupInternal}

// Name returns the group's human name.
func (g Group) Name() string {
	switch g {
	case GroupSyntax:
		return "Syntax"
	case GroupBinding:
		return "Binding"
	case GroupType:
		return "Type"
	case GroupRuntime:
		return "Runtime"
	case GroupInternal:
		return "Internal"
	default:
		return string(g)
	}
}

// Description explains when a group's codes are reported.
func (g Group) Description() string {
	switch g {
	case GroupSyntax:
		return "The source could not be read as Flux. Reported before anything is checked or run."
	case GroupBinding:
		return "A name does not resolve, or is declared more than once. Reported while checking, " +
			"or while running for a name that only some paths define."
	case GroupType:
		return "A value does not have the type its use requires. Reported while checking, and " +
			"downgraded to a warning or suppressed entirely by the configured mode."
	case GroupRuntime:
		return "The program was accepted and then failed while running. Reported identically by " +
			"the interpreter, the VM, and a generated executable."
	case GroupInternal:
		return "A defect in Flux itself. A program should not be able to provoke one; if yours " +
			"does, that is worth reporting."
	default:
		return ""
	}
}

// Code identifies one kind of diagnostic, for example "T_ARGUMENT_TYPE". Codes
// are part of the tool's contract: tests assert on them, editors filter by
// them, and users search for them, so a code's meaning must not drift even when
// its message is rewritten.
//
// The format is GROUP_NAME, where GROUP is one of [Groups] and NAME is an
// uppercase, underscore-separated description.
type Code string

// Group returns the code's reserved group, and false when the code does not
// carry one.
func (c Code) Group() (Group, bool) {
	prefix, _, found := strings.Cut(string(c), "_")
	if !found {
		return "", false
	}
	for _, group := range Groups {
		if Group(prefix) == group {
			return group, true
		}
	}
	return "", false
}

// String returns the code as written.
func (c Code) String() string { return string(c) }

// Label is a secondary location that explains a diagnostic: the declaration a
// mismatched argument refers back to, or the branch whose type disagrees with
// another.
type Label struct {
	Span    source.Span
	Message string
}

// Diagnostic is one reported problem. The zero value is not meaningful; build
// diagnostics with [Error], [Warning], or [New].
type Diagnostic struct {
	// Code is the stable identifier of this kind of problem.
	Code Code
	// Severity is how this diagnostic affects the command's outcome.
	Severity Severity
	// Message is the one-line description, lowercase and without a trailing
	// period, in the style of Go error strings.
	Message string
	// Primary is where the problem is. It points at the offending construct
	// itself, not at the construct that gives it meaning: an argument of the
	// wrong type is reported at the argument, with the parameter as related
	// information.
	Primary source.Span
	// Related are secondary locations that explain the primary one.
	Related []Label
	// Notes are additional remarks, rendered after the snippet.
	Notes []string
	// Help is an optional suggested fix.
	Help string
}

// New returns a diagnostic with the given code, severity, and location.
func New(code Code, severity Severity, primary source.Span, format string, args ...any) Diagnostic {
	return Diagnostic{
		Code:     code,
		Severity: severity,
		Message:  fmt.Sprintf(format, args...),
		Primary:  primary,
	}
}

// Error returns an error-severity diagnostic.
func Error(code Code, primary source.Span, format string, args ...any) Diagnostic {
	return New(code, SeverityError, primary, format, args...)
}

// Warning returns a warning-severity diagnostic.
func Warning(code Code, primary source.Span, format string, args ...any) Diagnostic {
	return New(code, SeverityWarning, primary, format, args...)
}

// WithRelated returns a copy carrying an additional related location.
func (d Diagnostic) WithRelated(span source.Span, format string, args ...any) Diagnostic {
	d.Related = appendCopy(d.Related, Label{Span: span, Message: fmt.Sprintf(format, args...)})
	return d
}

// WithNote returns a copy carrying an additional note.
func (d Diagnostic) WithNote(format string, args ...any) Diagnostic {
	d.Notes = appendCopy(d.Notes, fmt.Sprintf(format, args...))
	return d
}

// WithHelp returns a copy carrying a suggested fix, replacing any existing one.
func (d Diagnostic) WithHelp(format string, args ...any) Diagnostic {
	d.Help = fmt.Sprintf(format, args...)
	return d
}

// WithSeverity returns a copy at a different severity. Configuration modes use
// this to downgrade errors without touching codes, spans, or messages.
func (d Diagnostic) WithSeverity(severity Severity) Diagnostic {
	d.Severity = severity
	return d
}

// HasLocation reports whether the diagnostic points at a source. A diagnostic
// without one can still be rendered, just without a snippet.
func (d Diagnostic) HasLocation() bool { return d.Primary.IsValid() }

// appendCopy appends to a copy of the slice. Diagnostics are passed by value
// and built by chaining, so two chains that branch from one diagnostic must not
// end up sharing an array.
func appendCopy[T any](existing []T, value T) []T {
	updated := make([]T, len(existing), len(existing)+1)
	copy(updated, existing)
	return append(updated, value)
}

// ToolError is a failure outside any Flux program: an unreadable file, a
// missing Go toolchain, a malformed configuration file. It never carries a span
// because there is no Flux source to point at, and keeping it a distinct type
// stops the CLI from presenting an environment problem as a language error.
//
// The proposed exit convention treats a ToolError as exit status 2, separate
// from the status 1 used for language and check failures.
type ToolError struct {
	// Op names what failed, such as "read source file".
	Op string
	// Path is the file involved, if any.
	Path string
	// Err is the underlying cause.
	Err error
}

// Tool returns a ToolError for a failed operation on a path.
func Tool(op, path string, err error) *ToolError {
	return &ToolError{Op: op, Path: path, Err: err}
}

// Error formats a tool operation with its optional path and underlying cause.
func (e *ToolError) Error() string {
	switch {
	case e.Path != "" && e.Err != nil:
		return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
	case e.Path != "":
		return fmt.Sprintf("%s %s", e.Op, e.Path)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	default:
		return e.Op
	}
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *ToolError) Unwrap() error { return e.Err }

// Ref identifies a diagnostic inside a [Bag] so that later diagnostics can name
// it as their cause. The zero value, NoCause, means "not caused by anything
// already reported".
type Ref int

// NoCause is the zero Ref.
const NoCause Ref = 0

type entry struct {
	diagnostic Diagnostic
	cause      Ref
	order      int
}

// Bag collects the diagnostics of one compilation and decides which of them are
// worth showing.
//
// It exists because a single mistake produces a cascade: one undefined
// identifier makes its enclosing call, its enclosing expression, and the
// declaration around it all unresolvable. A producer records those follow-on
// problems with [Bag.AddCausedBy], and the bag reports only the root, keeping
// the consequences available for tools that want them.
//
// A Bag is not safe for concurrent use; each analysis owns its own.
type Bag struct {
	entries []entry
	seen    map[key]Ref
}

type key struct {
	code     Code
	severity Severity
	message  string
	primary  source.Span
}

// Add records a diagnostic and returns a reference to it. Adding a diagnostic
// that is already present, meaning it matches an existing one in code,
// severity, message, and primary span, returns the existing reference instead
// of duplicating it: the checker can reach the same node twice, and reporting
// one problem twice is never useful. An independent report also makes a
// previously suppressed consequence reportable.
func (b *Bag) Add(diagnostic Diagnostic) Ref {
	return b.AddCausedBy(diagnostic, NoCause)
}

// AddCausedBy records a diagnostic that exists only because of an earlier one.
// The result is stored but withheld from [Bag.All], so the user sees the
// mistake they made rather than its consequences.
func (b *Bag) AddCausedBy(diagnostic Diagnostic, cause Ref) Ref {
	identity := key{
		code:     diagnostic.Code,
		severity: diagnostic.Severity,
		message:  diagnostic.Message,
		primary:  diagnostic.Primary,
	}
	if existing, ok := b.seen[identity]; ok {
		if cause == NoCause {
			b.entries[int(existing)-1].cause = NoCause
		}
		return existing
	}
	if b.seen == nil {
		b.seen = make(map[key]Ref)
	}
	b.entries = append(b.entries, entry{diagnostic: diagnostic, cause: cause, order: len(b.entries)})
	ref := Ref(len(b.entries))
	b.seen[identity] = ref
	return ref
}

// Extend records several independent diagnostics.
func (b *Bag) Extend(diagnostics []Diagnostic) {
	for _, diagnostic := range diagnostics {
		b.Add(diagnostic)
	}
}

// All returns the diagnostics worth reporting, in a deterministic order:
// suppressed consequences are omitted, and the rest are sorted by source
// position, then by code, then by the order they were added. Two runs over the
// same program therefore produce byte-identical output.
func (b *Bag) All() []Diagnostic { return b.collect(false) }

// AllIncludingSuppressed returns every recorded diagnostic, including the
// consequences withheld by [Bag.All]. An editor that lets a user opt into the
// full cascade reads this; ordinary output does not.
func (b *Bag) AllIncludingSuppressed() []Diagnostic { return b.collect(true) }

// collect selects reportable entries, optionally includes suppressed
// consequences, and sorts a copy for stable output.
func (b *Bag) collect(includeSuppressed bool) []Diagnostic {
	selected := make([]entry, 0, len(b.entries))
	for _, e := range b.entries {
		if includeSuppressed || e.cause == NoCause {
			selected = append(selected, e)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left, right := selected[i], selected[j]
		if order := source.Compare(left.diagnostic.Primary, right.diagnostic.Primary); order != 0 {
			return order < 0
		}
		if left.diagnostic.Code != right.diagnostic.Code {
			return left.diagnostic.Code < right.diagnostic.Code
		}
		return left.order < right.order
	})
	ordered := make([]Diagnostic, len(selected))
	for i, e := range selected {
		ordered[i] = e.diagnostic
	}
	return ordered
}

// Len returns the number of reportable diagnostics.
func (b *Bag) Len() int { return len(b.All()) }

// Empty reports whether anything is worth reporting.
func (b *Bag) Empty() bool { return b.Len() == 0 }

// HasErrors reports whether any reportable diagnostic is an error.
func (b *Bag) HasErrors() bool { return b.has(SeverityError) }

// HasWarnings reports whether any reportable diagnostic is a warning.
func (b *Bag) HasWarnings() bool { return b.has(SeverityWarning) }

// has reports whether an unsuppressed entry has the requested severity.
func (b *Bag) has(severity Severity) bool {
	for _, e := range b.entries {
		if e.cause == NoCause && e.diagnostic.Severity == severity {
			return true
		}
	}
	return false
}

// WithSeverity returns the reportable diagnostics of one severity.
func (b *Bag) WithSeverity(severity Severity) []Diagnostic {
	var matching []Diagnostic
	for _, diagnostic := range b.All() {
		if diagnostic.Severity == severity {
			matching = append(matching, diagnostic)
		}
	}
	return matching
}

// Exit statuses, defined once so that the CLI and every generated executable
// agree on what a status means.
//
// The distinction that matters is between the last two. A program that fails to
// compile or to run has told the user something true about their program, and
// scripts treat that as a normal outcome. A tool failure means Flux could not
// do its job at all — an unreadable file, a missing Go toolchain, a defect in
// Flux — and a script that retries or reports differently needs to tell them
// apart without parsing messages.
const (
	// ExitSuccess means the command did what was asked.
	ExitSuccess = 0
	// ExitFailure means the program was rejected or failed while running.
	ExitFailure = 1
	// ExitToolFailure means Flux could not run the command: bad usage, a
	// missing file, a broken environment, or an internal defect.
	ExitToolFailure = 2
)
