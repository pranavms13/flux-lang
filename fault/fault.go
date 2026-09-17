// Package fault is the catalogue of failures a Flux program can produce while
// it runs.
//
// The interpreter and the VM are separate implementations of the same
// language, and a user is entitled to the same answer from both. Sharing one
// catalogue is what makes that true by construction: a failure has one code and
// one wording regardless of which engine noticed it, and a test can assert that
// the two agree without comparing prose.
//
// The package is part of the bundle a generated executable is built from, so it
// depends only on source and diagnostic, which have no dependencies of their
// own.
package fault

import (
	"fmt"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/source"
)

// The catalogue. Every failure either engine can report has an entry here.
var (
	CodeUndefinedValue = diagnostic.Register("R_UNDEFINED_VALUE",
		"a name was evaluated that nothing has bound")
	CodeNotCallable = diagnostic.Register("R_NOT_CALLABLE",
		"a value that is not a function was called")
	CodeArgumentCount = diagnostic.Register("R_ARGUMENT_COUNT",
		"a function was called with the wrong number of arguments")
	CodeOperandType = diagnostic.Register("R_OPERAND_TYPE",
		"an operator was applied to values it does not accept")
	CodeNotIndexable = diagnostic.Register("R_NOT_INDEXABLE",
		"a value that is not a list or dictionary was indexed")
	CodeIndexType = diagnostic.Register("R_INDEX_TYPE",
		"a list was indexed with something other than an int")
	CodeIndexRange = diagnostic.Register("R_INDEX_RANGE",
		"a list index is outside the list")
	CodeMissingKey = diagnostic.Register("R_MISSING_KEY",
		"a dictionary has no entry for the key it was given")
)

// Frame is one entry of a call trace: the function that was running and the
// place the call to it was written.
//
// The call site is the useful half. A trace of function names says which
// functions were involved; the call sites say which line of the program put
// them there.
type Frame struct {
	// Function labels the function being run, such as a name it was bound to,
	// or "<anonymous>" for a literal that was never named.
	Function string
	// Call is where the call was written.
	Call source.Location
}

// Error is a failure raised while a program runs.
//
// It is an ordinary Go error, so it travels back through both engines by
// return rather than by panic, and it carries the structured diagnostic that
// the renderer needs.
type Error struct {
	Code diagnostic.Code
	// Message is the one-line description, without a position.
	Message string
	// Where is the construct that failed. It is the zero Location when the
	// failing instruction has no recorded position.
	Where source.Location
	// Trace is the call stack at the point of failure, innermost first.
	Trace []Frame
}

func (e *Error) Error() string {
	if !e.Where.IsValid() {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Where, e.Message)
}

// Diagnostic renders the failure as a diagnostic record, so a runtime failure
// reaches the terminal, JSON, and editor outputs the same way a type error
// does.
func (e *Error) Diagnostic() diagnostic.Diagnostic {
	d := diagnostic.Error(e.Code, e.span(), "%s", e.Message)
	for _, frame := range e.Trace {
		if frame.Call.IsValid() {
			d = d.WithNote("in %s, called at %s", frame.Function, frame.Call)
		}
	}
	return d
}

func (e *Error) span() source.Span {
	if !e.Where.IsValid() {
		return source.NoSpan
	}
	// The span is rebuilt for a consumer that still has the source. The
	// identifier is the first source, which is the only one a single-file
	// program has; a multi-source build resolves it from the location's file.
	return source.Span{SourceID: 1, Start: e.Where.Start, End: e.Where.End}
}

// WithTrace returns a copy of the failure carrying a call trace. An engine
// attaches the trace as the failure unwinds, so the innermost frame is added
// first.
func (e *Error) WithTrace(trace []Frame) *Error {
	if len(trace) == 0 {
		return e
	}
	combined := make([]Frame, 0, len(e.Trace)+len(trace))
	combined = append(combined, e.Trace...)
	combined = append(combined, trace...)
	copied := *e
	copied.Trace = combined
	return &copied
}

// At returns a copy of the failure located at the given place. An engine calls
// it when it knows where the failing instruction came from but the catalogue
// entry was built without that knowledge.
func (e *Error) At(where source.Location) *Error {
	if e.Where.IsValid() || !where.IsValid() {
		return e
	}
	copied := *e
	copied.Where = where
	return &copied
}

func newError(code diagnostic.Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// UndefinedValue reports a name that nothing has bound.
func UndefinedValue(name string) *Error {
	return newError(CodeUndefinedValue, "undefined variable: %s", name)
}

// NotCallable reports a call of something that is not a function.
func NotCallable(value any) *Error {
	return newError(CodeNotCallable, "cannot call %s", describe(value))
}

// ArgumentCount reports a call with the wrong number of arguments.
func ArgumentCount(function string, want, got int) *Error {
	return newError(CodeArgumentCount, "%s expects %d %s, got %d",
		function, want, plural(want, "argument"), got)
}

// OperandType reports an operator applied to values it does not accept.
func OperandType(operator string, left, right any) *Error {
	return newError(CodeOperandType, "cannot apply %s to %s and %s",
		operator, describe(left), describe(right))
}

// NotIndexable reports indexing something that is neither list nor dictionary.
func NotIndexable(value any) *Error {
	return newError(CodeNotIndexable, "cannot index into %s", describe(value))
}

// IndexType reports a list index that is not an int.
func IndexType(index any) *Error {
	return newError(CodeIndexType, "list index must be int, got %s", describe(index))
}

// IndexRange reports a list index outside the list.
func IndexRange(index, length int) *Error {
	return newError(CodeIndexRange, "list index %d is out of range, the list has %d %s",
		index, length, plural(length, "element"))
}

// MissingKey reports a dictionary lookup that found nothing.
func MissingKey(key any) *Error {
	return newError(CodeMissingKey, "dictionary has no key %s", render(key))
}

// Anonymous is the label for a function literal that was never bound to a name.
const Anonymous = "<anonymous>"

// describe names a value's type in the language's own vocabulary. A user wrote
// Flux, so a failure about their program must not answer in Go's type names.
func describe(value any) string {
	switch v := value.(type) {
	case nil:
		return "void"
	case int:
		return "int"
	case string:
		return "string"
	case bool:
		return "bool"
	case []any:
		return "list"
	case map[any]any:
		return "dictionary"
	default:
		if named, ok := value.(interface{ FluxTypeName() string }); ok {
			return named.FluxTypeName()
		}
		return fmt.Sprintf("value of type %T", v)
	}
}

// render shows a value the way print would, quoting strings so that a missing
// key is distinguishable from a missing number.
func render(value any) string {
	if text, ok := value.(string); ok {
		return fmt.Sprintf("%q", text)
	}
	return fmt.Sprintf("%v", value)
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}
