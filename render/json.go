package render

import (
	"encoding/json"

	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/source"
)

// JSONVersion is the version of the JSON diagnostic representation.
//
// It is emitted on every document so that a consumer can refuse a shape it does
// not understand rather than silently misreading it. Adding a field does not
// change it; removing or repurposing one does.
const JSONVersion = 1

// jsonDocument is the top-level shape. Diagnostics are wrapped rather than
// emitted as a bare array so that the version has somewhere to live.
type jsonDocument struct {
	Version     int              `json:"version"`
	Diagnostics []jsonDiagnostic `json:"diagnostics"`
}

type jsonDiagnostic struct {
	Code     string        `json:"code"`
	Severity string        `json:"severity"`
	Message  string        `json:"message"`
	Primary  *jsonLocation `json:"primary,omitempty"`
	Related  []jsonLabel   `json:"related,omitempty"`
	Notes    []string      `json:"notes,omitempty"`
	Help     string        `json:"help,omitempty"`
	Trace    []jsonFrame   `json:"trace,omitempty"`
}

// jsonLocation reports both units on purpose. Byte offsets are what the
// compiler works in; line and column are what a person reads. A consumer that
// guesses one from the other gets it wrong on any line with a tab or a
// multi-byte character.
type jsonLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
}

type jsonLabel struct {
	Message  string        `json:"message"`
	Location *jsonLocation `json:"location,omitempty"`
}

type jsonFrame struct {
	Function string        `json:"function"`
	Call     *jsonLocation `json:"call,omitempty"`
}

// JSON renders diagnostics as a versioned JSON document.
func (r Renderer) JSON(diagnostics []diagnostic.Diagnostic) ([]byte, error) {
	document := jsonDocument{Version: JSONVersion, Diagnostics: make([]jsonDiagnostic, 0, len(diagnostics))}
	for _, d := range diagnostics {
		document.Diagnostics = append(document.Diagnostics, r.jsonOf(d))
	}
	return json.MarshalIndent(document, "", "  ")
}

// JSONFailure renders a runtime failure as the same document shape, so a
// consumer parses one format regardless of which stage failed.
func (r Renderer) JSONFailure(failure *fault.Error) ([]byte, error) {
	entry := jsonDiagnostic{
		Code:     failure.Code.String(),
		Severity: diagnostic.SeverityError.String(),
		Message:  failure.Message,
		Primary:  locationJSON(failure.Where),
	}
	for _, frame := range failure.Trace {
		entry.Trace = append(entry.Trace, jsonFrame{
			Function: frame.Function,
			Call:     locationJSON(frame.Call),
		})
	}
	return json.MarshalIndent(jsonDocument{
		Version:     JSONVersion,
		Diagnostics: []jsonDiagnostic{entry},
	}, "", "  ")
}

// jsonOf converts a diagnostic to the JSON schema, resolving only spans
// belonging to the configured source.
func (r Renderer) jsonOf(d diagnostic.Diagnostic) jsonDiagnostic {
	entry := jsonDiagnostic{
		Code:     d.Code.String(),
		Severity: d.Severity.String(),
		Message:  d.Message,
		Primary:  locationJSON(r.resolve(d.Primary)),
		Notes:    d.Notes,
		Help:     d.Help,
	}
	for _, related := range d.Related {
		entry.Related = append(entry.Related, jsonLabel{
			Message:  related.Message,
			Location: locationJSON(r.resolve(related.Span)),
		})
	}
	return entry
}

// resolve locates a span in the configured snapshot, or returns an unknown
// location when the source does not match.
func (r Renderer) resolve(span source.Span) source.Location {
	if r.Source == nil || !span.IsValid() || span.SourceID != r.Source.ID() {
		return source.Location{}
	}
	return r.Source.Locate(span)
}

// locationJSON converts a valid location to JSON fields, returning nil so
// unavailable locations are omitted.
func locationJSON(location source.Location) *jsonLocation {
	if !location.IsValid() {
		return nil
	}
	return &jsonLocation{
		File:   location.File,
		Line:   location.Line,
		Column: location.Column,
		Start:  location.Start,
		End:    location.End,
	}
}
