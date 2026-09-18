package source_test

import (
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/source"
)

// TestPositionCountsEachColumnUnitSeparately distinguishes byte, rune, UTF-16,
// and tab-expanded display columns.
func TestPositionCountsEachColumnUnitSeparately(t *testing.T) {
	// The line mixes a combining mark (é as e + U+0301), a supplementary-plane
	// emoji, and a tab so that every column unit disagrees.
	const line = "let é = \"\U0001F600\"\tx"
	src := source.New(1, "unicode.flux", line)

	for _, test := range []struct {
		name   string
		offset int
		want   source.Position
	}{
		{"start of file", 0, source.Position{Line: 1, Byte: 1, Rune: 1, UTF16: 1, Display: 1}},
		{"before combining mark", 4, source.Position{Line: 1, Byte: 5, Rune: 5, UTF16: 5, Display: 5}},
		// The combining mark is two bytes and one code point of its own.
		{"after combining mark", 7, source.Position{Line: 1, Byte: 8, Rune: 7, UTF16: 7, Display: 7}},
		// The emoji is four bytes, one rune, and two UTF-16 code units.
		{"after emoji", 15, source.Position{Line: 1, Byte: 16, Rune: 12, UTF16: 13, Display: 12}},
		// The tab at display column 13 advances to the next stop, column 17.
		{"after tab", 17, source.Position{Line: 1, Byte: 18, Rune: 14, UTF16: 15, Display: 17}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := src.Position(test.offset); got != test.want {
				t.Errorf("Position(%d) = %+v, want %+v", test.offset, got, test.want)
			}
		})
	}
}

// TestPositionInsideRuneReportsThatRunesColumn checks that offsets inside a
// multibyte rune resolve to its starting column.
func TestPositionInsideRuneReportsThatRunesColumn(t *testing.T) {
	src := source.New(1, "unicode.flux", "\U0001F600x")
	// Offsets 0 through 3 all fall within the emoji; none of them may report a
	// column past it, which would place a caret on the following character.
	for offset := 0; offset < 4; offset++ {
		if got := src.Position(offset); got.Rune != 1 {
			t.Errorf("Position(%d).Rune = %d, want 1", offset, got.Rune)
		}
	}
	if got := src.Position(4).Rune; got != 2 {
		t.Errorf("Position(4).Rune = %d, want 2", got)
	}
}

// TestTabExpansionHonorsStopWidth checks custom tab stops and the default used
// for invalid widths.
func TestTabExpansionHonorsStopWidth(t *testing.T) {
	src := source.New(1, "tabs.flux", "\tab\tc")
	for _, test := range []struct {
		tabWidth, offset, want int
	}{
		{source.DefaultTabWidth, 0, 1}, // before the leading tab
		{source.DefaultTabWidth, 1, 9}, // the leading tab fills columns 1-8
		{source.DefaultTabWidth, 3, 11},
		{source.DefaultTabWidth, 4, 17}, // second tab advances from 11 to 17
		{4, 1, 5},
		{4, 4, 9},
		{1, 1, 2},
		{0, 1, 2}, // a width below one still advances one cell
	} {
		got := src.PositionWithTabWidth(test.offset, test.tabWidth).Display
		if got != test.want {
			t.Errorf("PositionWithTabWidth(%d, tab=%d).Display = %d, want %d",
				test.offset, test.tabWidth, got, test.want)
		}
	}
}

// TestCRLFLinesExcludeTheTerminator verifies line text and positions across
// CRLF boundaries.
func TestCRLFLinesExcludeTheTerminator(t *testing.T) {
	src := source.New(1, "crlf.flux", "let a = 1\r\nlet b = 2\r\n")

	if got, want := src.LineCount(), 3; got != want {
		t.Errorf("LineCount() = %d, want %d (trailing newline opens a final empty line)", got, want)
	}
	if got, want := src.LineText(1), "let a = 1"; got != want {
		t.Errorf("LineText(1) = %q, want %q", got, want)
	}
	if got, want := src.LineText(3), ""; got != want {
		t.Errorf("LineText(3) = %q, want %q", got, want)
	}
	// The carriage return belongs to the line it terminates, not the next one.
	if got, want := src.Position(9), (source.Position{Line: 1, Byte: 10, Rune: 10, UTF16: 10, Display: 10}); got != want {
		t.Errorf("Position(9) = %+v, want %+v", got, want)
	}
	if got, want := src.Position(11).Line, 2; got != want {
		t.Errorf("Position(11).Line = %d, want %d", got, want)
	}
}

// TestEmptySourceHasOneLineAndOnePosition pins line, position, and EOF
// behavior for an empty snapshot.
func TestEmptySourceHasOneLineAndOnePosition(t *testing.T) {
	src := source.New(1, "empty.flux", "")

	if got, want := src.LineCount(), 1; got != want {
		t.Errorf("LineCount() = %d, want %d", got, want)
	}
	want := source.Position{Line: 1, Byte: 1, Rune: 1, UTF16: 1, Display: 1}
	if got := src.Position(0); got != want {
		t.Errorf("Position(0) = %+v, want %+v", got, want)
	}
	if got := src.Position(50); got != want {
		t.Errorf("Position(50) = %+v, want %+v (out-of-range offsets clamp)", got, want)
	}
	if got := src.EOF(); got != (source.Span{SourceID: 1}) {
		t.Errorf("EOF() = %+v, want an empty span at offset 0", got)
	}
	if got, want := src.LineText(1), ""; got != want {
		t.Errorf("LineText(1) = %q, want %q", got, want)
	}
}

// TestEOFSpanPointsPastTheLastCharacter checks that EOF is an empty span at
// the source length.
func TestEOFSpanPointsPastTheLastCharacter(t *testing.T) {
	const text = "let a = 1\nlet b ="
	src := source.New(1, "eof.flux", text)

	eof := src.EOF()
	if !eof.IsEmpty() || eof.Start != len(text) {
		t.Fatalf("EOF() = %+v, want an empty span at offset %d", eof, len(text))
	}
	want := source.Position{Line: 2, Byte: 8, Rune: 8, UTF16: 8, Display: 8}
	if got := src.Position(eof.Start); got != want {
		t.Errorf("Position at EOF = %+v, want %+v", got, want)
	}
	if got, want := src.TextOf(eof), ""; got != want {
		t.Errorf("TextOf(EOF) = %q, want %q", got, want)
	}
}

// TestMultilineSpanCoversEveryLineItTouches checks extraction and endpoint
// positions for a range crossing a newline.
func TestMultilineSpanCoversEveryLineItTouches(t *testing.T) {
	const text = "let f = fn(a) =>\n  a + 1\nf(2)\n"
	src := source.New(1, "multiline.flux", text)

	span := src.Span(strings.Index(text, "fn"), strings.Index(text, "a + 1")+len("a + 1"))
	if got, want := src.TextOf(span), "fn(a) =>\n  a + 1"; got != want {
		t.Errorf("TextOf(span) = %q, want %q", got, want)
	}
	start, end := src.Position(span.Start), src.Position(span.End)
	if start.Line != 1 || end.Line != 2 {
		t.Errorf("span runs from line %d to line %d, want 1 to 2", start.Line, end.Line)
	}
	if got, want := end.Display, 8; got != want {
		t.Errorf("end display column = %d, want %d", got, want)
	}
}

// TestSpanConstructionClampsAndOrders checks that reversed or out-of-range
// endpoints produce bounded, ordered spans.
func TestSpanConstructionClampsAndOrders(t *testing.T) {
	src := source.New(1, "clamp.flux", "abc")

	if got, want := src.Span(2, 0), (source.Span{SourceID: 1, Start: 0, End: 2}); got != want {
		t.Errorf("Span(2, 0) = %+v, want %+v (reversed bounds are ordered)", got, want)
	}
	if got, want := src.Span(-5, 99), (source.Span{SourceID: 1, Start: 0, End: 3}); got != want {
		t.Errorf("Span(-5, 99) = %+v, want %+v", got, want)
	}
	if got, want := src.Whole(), (source.Span{SourceID: 1, Start: 0, End: 3}); got != want {
		t.Errorf("Whole() = %+v, want %+v", got, want)
	}
}

// TestSpanPredicates checks validity, half-open containment, and overlap
// within a single source.
func TestSpanPredicates(t *testing.T) {
	a := source.Span{SourceID: 1, Start: 4, End: 8}
	empty := source.Span{SourceID: 1, Start: 4, End: 4}

	if source.NoSpan.IsValid() {
		t.Error("NoSpan.IsValid() = true, want false")
	}
	if (source.Span{SourceID: 1, Start: 5, End: 2}).IsValid() {
		t.Error("a reversed span reports valid")
	}
	if !a.IsValid() || a.Len() != 4 || a.IsEmpty() {
		t.Errorf("%+v: want a valid, non-empty span of length 4", a)
	}
	if !empty.IsEmpty() || empty.Contains(4) {
		t.Error("an empty span must contain nothing")
	}
	if !a.Contains(4) || !a.Contains(7) || a.Contains(8) {
		t.Error("Contains must treat the span as half-open")
	}
	if !a.Overlaps(source.Span{SourceID: 1, Start: 7, End: 9}) {
		t.Error("overlapping spans report no overlap")
	}
	if a.Overlaps(source.Span{SourceID: 2, Start: 4, End: 8}) {
		t.Error("spans in different sources must never overlap")
	}
	if a.Overlaps(source.Span{SourceID: 1, Start: 8, End: 9}) {
		t.Error("adjacent spans must not overlap")
	}
}

// TestSpanUnion checks merged ranges, invalid spans, and rejection of spans
// from different sources.
func TestSpanUnion(t *testing.T) {
	a := source.Span{SourceID: 1, Start: 4, End: 8}
	b := source.Span{SourceID: 1, Start: 12, End: 14}

	if got, want := a.Union(b), (source.Span{SourceID: 1, Start: 4, End: 14}); got != want {
		t.Errorf("a.Union(b) = %+v, want %+v", got, want)
	}
	if got := a.Union(source.Span{SourceID: 2, Start: 0, End: 1}); got != source.NoSpan {
		t.Errorf("union across sources = %+v, want NoSpan", got)
	}
	if got := a.Union(source.NoSpan); got != a {
		t.Errorf("a.Union(NoSpan) = %+v, want %+v", got, a)
	}
	if got := source.NoSpan.Union(a); got != a {
		t.Errorf("NoSpan.Union(a) = %+v, want %+v", got, a)
	}
}

// TestCompareOrdersBySourceThenStartThenEnd pins the ordering used for
// deterministic diagnostics.
func TestCompareOrdersBySourceThenStartThenEnd(t *testing.T) {
	for _, test := range []struct {
		name string
		a, b source.Span
		want int
	}{
		{"earlier source", source.Span{SourceID: 1}, source.Span{SourceID: 2}, -1},
		{"earlier start", source.Span{SourceID: 1, Start: 1}, source.Span{SourceID: 1, Start: 3}, -1},
		{"shorter span first", source.Span{SourceID: 1, End: 2}, source.Span{SourceID: 1, End: 5}, -1},
		{"identical", source.Span{SourceID: 1, Start: 2, End: 5}, source.Span{SourceID: 1, Start: 2, End: 5}, 0},
		{"later start", source.Span{SourceID: 1, Start: 9}, source.Span{SourceID: 1, Start: 3}, 1},
	} {
		if got := source.Compare(test.a, test.b); got != test.want {
			t.Errorf("%s: Compare = %d, want %d", test.name, got, test.want)
		}
	}
}

// TestLineRangeIncludesTerminatorAndRejectsMissingLines checks raw line
// boundaries and invalid line requests.
func TestLineRangeIncludesTerminatorAndRejectsMissingLines(t *testing.T) {
	src := source.New(1, "lines.flux", "ab\ncd")

	start, end, ok := src.LineRange(1)
	if !ok || start != 0 || end != 3 {
		t.Errorf("LineRange(1) = (%d, %d, %v), want (0, 3, true)", start, end, ok)
	}
	if _, _, ok := src.LineRange(0); ok {
		t.Error("LineRange(0) reported a line; lines are 1-based")
	}
	if _, _, ok := src.LineRange(3); ok {
		t.Error("LineRange(3) reported a line that does not exist")
	}
	if got := src.LineText(9); got != "" {
		t.Errorf("LineText(9) = %q, want %q", got, "")
	}
}

// TestTextOfRejectsSpansFromAnotherSource prevents offsets from one snapshot
// from extracting text from another.
func TestTextOfRejectsSpansFromAnotherSource(t *testing.T) {
	src := source.New(1, "a.flux", "abcdef")
	if got := src.TextOf(source.Span{SourceID: 2, Start: 0, End: 3}); got != "" {
		t.Errorf("TextOf(foreign span) = %q, want %q", got, "")
	}
	if got, want := src.TextOf(src.Span(1, 3)), "bc"; got != want {
		t.Errorf("TextOf = %q, want %q", got, want)
	}
}

// TestSnapshotExposesItsOriginalBytesAndName checks that source identity and
// original content are retained.
func TestSnapshotExposesItsOriginalBytesAndName(t *testing.T) {
	const text = "let a = 1\n"
	src := source.New(7, "display.flux", text)

	if got, want := src.ID(), source.SourceID(7); got != want {
		t.Errorf("ID() = %d, want %d", got, want)
	}
	if got, want := src.Name(), "display.flux"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if got := src.Text(); got != text {
		t.Errorf("Text() = %q, want %q", got, text)
	}
	if got, want := src.Len(), len(text); got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

// TestMapAssignsDistinctIdentifiers checks unique source IDs and lookup
// failure for unknown IDs.
func TestMapAssignsDistinctIdentifiers(t *testing.T) {
	m := source.NewMap()
	first := m.Add("a.flux", "let a = 1")
	second := m.Add("b.flux", "let b = 2")

	if first.ID() == second.ID() {
		t.Fatalf("both sources share ID %d", first.ID())
	}
	if first.ID() == source.NoSource {
		t.Error("a registered source was given the NoSource identifier")
	}
	if m.Len() != 2 {
		t.Errorf("Len() = %d, want 2", m.Len())
	}

	got, ok := m.Source(second.ID())
	if !ok || got != second {
		t.Errorf("Source(%d) did not return the registered snapshot", second.ID())
	}
	if _, ok := m.Source(source.NoSource); ok {
		t.Error("Source(NoSource) returned a snapshot")
	}
	if _, ok := m.Source(source.SourceID(99)); ok {
		t.Error("Source(99) returned a snapshot that was never added")
	}
}

// TestMapKeepsSnapshotsOfTheSameFileSeparate verifies that repeated filenames
// do not collapse distinct source revisions.
func TestMapKeepsSnapshotsOfTheSameFileSeparate(t *testing.T) {
	m := source.NewMap()
	before := m.Add("edit.flux", "let a = 1")
	after := m.Add("edit.flux", "let a = 2")

	if before.ID() == after.ID() {
		t.Fatal("two snapshots of one path share an identifier")
	}
	if before.Text() != "let a = 1" {
		t.Errorf("the earlier snapshot changed to %q", before.Text())
	}
}

// TestMapIsSafeForConcurrentUse checks unique IDs and consistent lookups while
// sources are added concurrently.
func TestMapIsSafeForConcurrentUse(t *testing.T) {
	m := source.NewMap()
	const writers = 8
	done := make(chan source.SourceID, writers)
	for i := 0; i < writers; i++ {
		go func() { done <- m.Add("concurrent.flux", "let a = 1").ID() }()
	}
	seen := make(map[source.SourceID]bool, writers)
	for i := 0; i < writers; i++ {
		id := <-done
		if seen[id] {
			t.Errorf("identifier %d was handed out twice", id)
		}
		seen[id] = true
		if _, ok := m.Source(id); !ok {
			t.Errorf("Source(%d) is missing after a concurrent Add", id)
		}
	}
}
