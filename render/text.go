// Package render turns diagnostics into something a person or a tool reads.
//
// It is the only component that decides what a diagnostic looks like. The
// parser, the checker, and both engines produce records; nothing else in the
// project writes a caret, a colour escape, or a JSON field name, so a
// diagnostic looks the same whichever of them found it.
//
// The package belongs to the bundle a generated executable is built from, so it
// depends only on source, diagnostic, and fault.
package render

import (
	"fmt"
	"os"
	"strings"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/source"
)

// DefaultTabWidth is the tab stop width used when a snippet is drawn. It has to
// match the width used to compute display columns, or the caret lands in the
// wrong place on any line containing a tab.
const DefaultTabWidth = source.DefaultTabWidth

// Renderer draws diagnostics for a terminal.
//
// The zero value is usable: no colour, no snippets, default tab width.
type Renderer struct {
	// Source is the program's text, used to draw the line a diagnostic points
	// at. It may be nil — an executable built without debug information has no
	// source — and the renderer then prints the position without the line.
	Source *source.Source
	// Color adds ANSI styling. It is off by default so that redirected output,
	// golden files, and CI logs are stable.
	Color bool
	// TabWidth is the tab stop width for snippets. Zero means DefaultTabWidth.
	TabWidth int
}

// Diagnostic renders one diagnostic, with a snippet when the source is
// available and the diagnostic points into it.
func (r Renderer) Diagnostic(d diagnostic.Diagnostic) string {
	var out strings.Builder
	out.WriteString(r.headline(r.locate(d.Primary), d.Severity, d.Code, d.Message))
	if snippet := r.snippet(d.Primary); snippet != "" {
		out.WriteString("\n")
		out.WriteString(snippet)
	}
	for _, related := range d.Related {
		out.WriteString("\n  = note: " + related.Message)
		if where := r.locate(related.Span); where != "" {
			out.WriteString(" at " + where)
		}
	}
	for _, note := range d.Notes {
		out.WriteString("\n  = note: " + note)
	}
	if d.Help != "" {
		out.WriteString("\n  = help: " + d.Help)
	}
	return out.String()
}

// Diagnostics renders several, one after another.
func (r Renderer) Diagnostics(diagnostics []diagnostic.Diagnostic) string {
	rendered := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		rendered = append(rendered, r.Diagnostic(d))
	}
	return strings.Join(rendered, "\n")
}

// Failure renders a runtime failure, with the calls that led to it.
//
// A failure carries positions that were resolved when the program was compiled,
// so this works in a generated executable whose source file is long gone.
func (r Renderer) Failure(failure *fault.Error) string {
	var out strings.Builder
	severity := diagnostic.SeverityError
	out.WriteString(r.headline(failure.Where.String(), severity, failure.Code, failure.Message))
	if failure.Where.IsValid() && r.Source != nil && r.Source.Name() == failure.Where.File {
		if snippet := r.snippetAt(failure.Where.Line, failure.Where.Start, failure.Where.End); snippet != "" {
			out.WriteString("\n")
			out.WriteString(snippet)
		}
	}
	for _, frame := range failure.Trace[:min(len(failure.Trace), fault.MaxTraceFrames)] {
		out.WriteString(fmt.Sprintf("\n  in %s, called at %s", frame.Function, frame.Call))
	}
	if len(failure.Trace) > fault.MaxTraceFrames {
		fmt.Fprintf(&out, "\n  ... %d more calls", len(failure.Trace)-fault.MaxTraceFrames)
	}
	if failure.Code == diagnostic.CodeInternal {
		out.WriteString("\n  = note: this is a bug in Flux, not in the program being run")
	}
	return out.String()
}

// ToolError renders a failure that has no place in any Flux source, such as an
// unreadable file. It deliberately looks different from a language error: the
// mistake is not in the program.
func (r Renderer) ToolError(err *diagnostic.ToolError) string {
	return r.style(severityColor(diagnostic.SeverityError), "error") + ": " + err.Error()
}

// headline formats the severity, code, and message, adding a location and
// internal-defect label when applicable.
func (r Renderer) headline(where string, severity diagnostic.Severity, code diagnostic.Code, message string) string {
	label := severity.String()
	if code == diagnostic.CodeInternal {
		label = "internal " + label
	}
	headline := r.style(severityColor(severity), fmt.Sprintf("%s[%s]", label, code)) + ": " + message
	if where == "" || where == "<unknown>" {
		return headline
	}
	return where + ": " + headline
}

// locate formats a span location only when it belongs to the configured
// source.
func (r Renderer) locate(span source.Span) string {
	if r.Source == nil || !span.IsValid() || span.SourceID != r.Source.ID() {
		return ""
	}
	return r.Source.Locate(span).String()
}

// snippet draws a span only when the configured source has its source
// identity.
func (r Renderer) snippet(span source.Span) string {
	if r.Source == nil || !span.IsValid() || span.SourceID != r.Source.ID() {
		return ""
	}
	return r.snippetAt(r.Source.Position(span.Start).Line, span.Start, span.End)
}

// snippetAt draws the line containing a byte range, with the range underlined.
//
// Tabs in the line are expanded before it is printed, and the underline is
// measured in the same expanded columns. Printing the line raw and counting
// columns separately is how carets end up in the wrong place.
func (r Renderer) snippetAt(line, start, end int) string {
	if r.Source == nil || line < 1 || line > r.Source.LineCount() {
		return ""
	}
	tabWidth := r.TabWidth
	if tabWidth < 1 {
		tabWidth = DefaultTabWidth
	}
	text := expandTabs(r.Source.LineText(line), tabWidth)
	if text == "" {
		return ""
	}

	from := r.Source.PositionWithTabWidth(start, tabWidth)
	to := r.Source.PositionWithTabWidth(end, tabWidth)
	width := 1
	if to.Line == from.Line && to.Display > from.Display {
		width = to.Display - from.Display
	} else if to.Line != from.Line {
		// The range continues past this line; underline the rest of it.
		width = len([]rune(text)) - from.Display + 1
	}
	if width < 1 {
		width = 1
	}

	number := fmt.Sprintf("%d", line)
	gutter := strings.Repeat(" ", len(number))
	underline := strings.Repeat(" ", from.Display-1) + r.style(colorCaret, strings.Repeat("^", width))
	return fmt.Sprintf("%s | %s\n%s | %s", number, text, gutter, underline)
}

// expandTabs replaces tabs with spaces up to the next tab stop, matching how
// source.Position counts display columns.
func expandTabs(text string, tabWidth int) string {
	if !strings.ContainsRune(text, '\t') {
		return text
	}
	var out strings.Builder
	column := 0
	for _, r := range text {
		if r == '\t' {
			spaces := tabWidth - column%tabWidth
			out.WriteString(strings.Repeat(" ", spaces))
			column += spaces
			continue
		}
		out.WriteRune(r)
		column++
	}
	return out.String()
}

const (
	colorError   = "\x1b[1;31m"
	colorWarning = "\x1b[1;33m"
	colorInfo    = "\x1b[1;36m"
	colorCaret   = "\x1b[1;31m"
	colorReset   = "\x1b[0m"
)

// severityColor selects the ANSI style for an error, warning, or informational
// diagnostic.
func severityColor(severity diagnostic.Severity) string {
	switch severity {
	case diagnostic.SeverityWarning:
		return colorWarning
	case diagnostic.SeverityInfo:
		return colorInfo
	default:
		return colorError
	}
}

// style wraps text in ANSI escapes only when color output is enabled.
func (r Renderer) style(color, text string) string {
	if !r.Color {
		return text
	}
	return color + text + colorReset
}

// ColorEnabled reports whether styling should be used for a destination.
//
// Colour is off unless the destination is a terminal, and off entirely when
// NO_COLOR is set, so redirected output and CI logs stay byte-stable without
// every caller having to remember to ask.
func ColorEnabled(w any) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	file, ok := w.(interface{ Stat() (os.FileInfo, error) })
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
