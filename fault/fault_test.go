package fault_test

import (
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/render"
	"github.com/pranavms13/flux-lang/source"
)

// TestDiagnosticPreservesSourceIdentity checks fault conversion against a map
// containing another source before the one that failed.
func TestDiagnosticPreservesSourceIdentity(t *testing.T) {
	var sources source.Map
	other := sources.Add("other.flux", "unrelated")
	src := sources.Add("failure.flux", "missing")
	failure := fault.UndefinedValue("missing").At(src.Locate(src.Whole()))
	d := failure.Diagnostic()
	if d.Primary != src.Whole() {
		t.Fatalf("primary = %+v, want %+v", d.Primary, src.Whole())
	}
	if got := (render.Renderer{Source: src}).Diagnostic(d); !strings.Contains(got, "1 | missing") {
		t.Error("the originating source cannot render the diagnostic snippet")
	}
	if got := (render.Renderer{Source: other}).Diagnostic(d); strings.Contains(got, "unrelated") {
		t.Error("the diagnostic was rendered against a different source")
	}
}

// TestDiagnosticWithoutSourceIdentity does not invent a span for a location
// that only carries display information, as older serialized programs can.
func TestDiagnosticWithoutSourceIdentity(t *testing.T) {
	for _, where := range []source.Location{
		{},
		{File: "legacy.flux", Line: 1, Column: 1, Start: 0, End: 7},
	} {
		d := fault.UndefinedValue("missing").At(where).Diagnostic()
		if d.Primary != source.NoSpan {
			t.Errorf("primary = %+v, want no span for location %+v", d.Primary, where)
		}
	}
}
