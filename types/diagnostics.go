package types

import (
	"fmt"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/source"
)

// Diagnostic codes reported by the checker.
//
// Most belong to the type group. Two belong to the binding group instead: an
// undefined name and a repeated parameter are mistakes about which names exist,
// not about what types they have, and the group is what tells a reader that
// before they look the code up.
var (
	CodeAnnotationMismatch = diagnostic.Register("T_ANNOTATION_MISMATCH",
		"a value does not have the type its declaration annotates")
	CodeInvalidAnnotation = diagnostic.Register("T_INVALID_ANNOTATION",
		"a type annotation does not name a type")
	CodeReturnMismatch = diagnostic.Register("T_RETURN_MISMATCH",
		"a function body does not produce the return type it declares")
	CodeConditionType = diagnostic.Register("T_CONDITION_TYPE",
		"the condition of a conditional is not a bool")
	CodeBranchMismatch = diagnostic.Register("T_BRANCH_MISMATCH",
		"the branches of a conditional produce different types")
	CodeOperandType = diagnostic.Register("T_OPERAND_TYPE",
		"an operator was applied to types it does not accept")
	CodeComparisonMismatch = diagnostic.Register("T_COMPARISON_MISMATCH",
		"two values of different types were compared")
	CodeListElementType = diagnostic.Register("T_LIST_ELEMENT_TYPE",
		"a list element does not have the type of the other elements")
	CodeDictKeyType = diagnostic.Register("T_DICT_KEY_TYPE",
		"a dictionary key does not have the type of the other keys")
	CodeDictValueType = diagnostic.Register("T_DICT_VALUE_TYPE",
		"a dictionary value does not have the type of the other values")
	CodeInvalidDictKey = diagnostic.Register("T_INVALID_DICT_KEY",
		"a dictionary key is not an int, string, or bool")
	CodeNotCallable = diagnostic.Register("T_NOT_CALLABLE",
		"a value that is not a function was called")
	CodeArgumentCount = diagnostic.Register("T_ARGUMENT_COUNT",
		"a function was called with the wrong number of arguments")
	CodeArgumentType = diagnostic.Register("T_ARGUMENT_TYPE",
		"an argument's type does not match the parameter it is passed to")
	CodeIndexType = diagnostic.Register("T_INDEX_TYPE",
		"a collection was indexed with the wrong type")
	CodeNotIndexable = diagnostic.Register("T_NOT_INDEXABLE",
		"a value that is not a list or dictionary was indexed")
	CodeUndefinedVariable = diagnostic.Register("B_UNDEFINED_VARIABLE",
		"a name was used where nothing declares it")
	CodeDuplicateParameter = diagnostic.Register("B_DUPLICATE_PARAMETER",
		"a function declares the same parameter name twice")
)

// policy says how the configured mode decides a diagnostic's severity.
type policy int

const (
	// always marks a mistake that is an error in every checking mode. Only
	// warn-only downgrades it.
	always policy = iota
	// strictOnly marks a mistake the language tolerates outside strict mode,
	// where it is reported as a warning and checking continues.
	strictOnly
)

// report records a diagnostic after deciding its severity.
//
// This function is the only place a configuration mode changes a severity.
// Codes, messages, spans and related information are identical in every mode,
// so a test can assert on the code while the mode decides only whether the
// command fails.
func (tc *TypeChecker) report(p policy, d diagnostic.Diagnostic) {
	severity := diagnostic.SeverityError
	if p == strictOnly && !tc.config.Strict {
		severity = diagnostic.SeverityWarning
	}
	if severity == diagnostic.SeverityError && tc.config.WarnOnly {
		severity = diagnostic.SeverityWarning
	}
	tc.diagnostics.Add(d.WithSeverity(severity))
}

// errorAt reports a mistake located at a node.
func (tc *TypeChecker) errorAt(p policy, code diagnostic.Code, at ast.Positioned, format string, args ...any) diagnostic.Diagnostic {
	d := diagnostic.Error(code, tc.span(at), format, args...)
	tc.report(p, d)
	return d
}

// span converts a node's position into a located span, or NoSpan when the
// checker was given no source.
func (tc *TypeChecker) span(at ast.Positioned) source.Span {
	if at == nil || tc.sourceID == source.NoSource {
		return source.NoSpan
	}
	return at.Span(tc.sourceID)
}

// reportWithRelated reports a mistake and attaches a second location that
// explains it, such as the parameter an argument is passed to. The label is
// dropped when that location is unknown, which happens for a built-in or for a
// function whose type came from an annotation rather than a literal.
func (tc *TypeChecker) reportWithRelated(p policy, code diagnostic.Code, at ast.Positioned,
	related source.Span, relatedFormat string, relatedArgs []any, format string, args ...any) {
	d := diagnostic.Error(code, tc.span(at), format, args...)
	if related.IsValid() {
		d = d.WithRelated(related, relatedFormat, relatedArgs...)
	}
	tc.report(p, d)
}

// Diagnostics returns everything the checker reported, in source order.
func (tc *TypeChecker) Diagnostics() []diagnostic.Diagnostic { return tc.diagnostics.All() }

// HasErrors reports whether the program failed checking.
func (tc *TypeChecker) HasErrors() bool { return tc.diagnostics.HasErrors() }

// HasWarnings reports whether the checker reported anything that does not fail
// the program.
func (tc *TypeChecker) HasWarnings() bool { return tc.diagnostics.HasWarnings() }

// GetErrors returns the error messages as strings.
//
// It is an adapter for callers that predate structured diagnostics. P1.6
// replaces it with a renderer that can show snippets and related locations;
// until then it renders one line each, with a position when the checker was
// given a source.
func (tc *TypeChecker) GetErrors() []string {
	return tc.render(diagnostic.SeverityError)
}

// GetWarnings returns the warning messages as strings. See [TypeChecker.GetErrors].
func (tc *TypeChecker) GetWarnings() []string {
	return tc.render(diagnostic.SeverityWarning)
}

// render formats reportable diagnostics of one severity for the legacy string
// accessors.
func (tc *TypeChecker) render(severity diagnostic.Severity) []string {
	diagnostics := tc.diagnostics.WithSeverity(severity)
	rendered := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		rendered = append(rendered, tc.renderOne(d))
	}
	return rendered
}

// renderOne appends explanatory notes and, when available, a source position
// to a diagnostic message.
func (tc *TypeChecker) renderOne(d diagnostic.Diagnostic) string {
	message := d.Message
	for _, note := range d.Notes {
		message += " (" + note + ")"
	}
	if tc.source == nil || !d.Primary.IsValid() {
		return message
	}
	position := tc.source.Position(d.Primary.Start)
	return fmt.Sprintf("%s:%d:%d: %s", tc.source.Name(), position.Line, position.Display, message)
}

// DiagnosticsWithSeverity returns the reported diagnostics of one severity, for
// a caller that renders errors and warnings differently.
func (tc *TypeChecker) DiagnosticsWithSeverity(severity diagnostic.Severity) []diagnostic.Diagnostic {
	return tc.diagnostics.WithSeverity(severity)
}
