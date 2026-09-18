// Package source models Flux source files and the byte ranges that locate
// constructs inside them.
//
// The package deliberately depends on nothing else in this module. The
// standalone executable bundle embeds the VM and its runtime-only
// dependencies, so anything the VM needs in order to report an error must be
// free of the parser, the type checker, and the AST.
//
// A location is canonically a half-open byte span [Start, End). Byte offsets
// are stable, cheap to produce from the parser, and unambiguous. They are not
// terminal columns and not LSP character offsets; both of those are derived on
// demand by [Source.Position], which counts bytes, runes, UTF-16 code units,
// and tab-expanded display cells separately.
package source

import (
	"fmt"
	"sort"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
)

// SourceID identifies one source within a [Map]. The zero value, NoSource,
// means "no source": a diagnostic carrying it has no location.
type SourceID uint32

// NoSource is the zero SourceID, used by spans that point at no file.
const NoSource SourceID = 0

// DefaultTabWidth is the tab stop width used when expanding tabs into display
// columns. Terminals conventionally use eight; renderers that expand tabs
// differently must pass their own width to [Source.PositionWithTabWidth].
const DefaultTabWidth = 8

// Source is an immutable snapshot of one Flux source file: its original bytes,
// the filename to display in diagnostics, and an index of line starts.
//
// A Source never changes after construction. A language server that receives an
// edit creates a new snapshot rather than mutating an existing one, so an
// in-flight analysis keeps reporting positions consistent with the text it read.
type Source struct {
	id   SourceID
	name string
	text string
	// lineStarts[i] is the byte offset at which line i+1 begins. It always
	// contains at least one element, so every source has a line 1.
	lineStarts []int
}

// New returns an immutable snapshot of text. The name is used for display only;
// it is never opened, so a caller may pass a label such as "<stdin>".
func New(id SourceID, name, text string) *Source {
	return &Source{id: id, name: name, text: text, lineStarts: indexLines(text)}
}

// indexLines records byte offsets after each newline, including the start of
// an empty final line.
func indexLines(text string) []int {
	starts := make([]int, 1, 1+len(text)/32)
	for offset := 0; offset < len(text); offset++ {
		if text[offset] == '\n' {
			starts = append(starts, offset+1)
		}
	}
	return starts
}

// ID returns the identifier this source was registered under.
func (s *Source) ID() SourceID { return s.id }

// Name returns the display filename.
func (s *Source) Name() string { return s.name }

// Text returns the original bytes as a string. Strings are immutable in Go, so
// this hands out the snapshot without copying it.
func (s *Source) Text() string { return s.text }

// Len returns the size of the source in bytes.
func (s *Source) Len() int { return len(s.text) }

// LineCount returns the number of lines. A source ending in a newline has a
// final empty line, matching how editors number a trailing blank line.
func (s *Source) LineCount() int { return len(s.lineStarts) }

// Span returns a span over this source, clamped to its bounds and ordered so
// that Start <= End.
func (s *Source) Span(start, end int) Span {
	if start > end {
		start, end = end, start
	}
	return Span{SourceID: s.id, Start: s.clamp(start), End: s.clamp(end)}
}

// Whole returns a span covering the entire source.
func (s *Source) Whole() Span { return Span{SourceID: s.id, Start: 0, End: len(s.text)} }

// EOF returns the empty span at the end of the source, which is where an
// unexpected end of input is reported.
func (s *Source) EOF() Span {
	return Span{SourceID: s.id, Start: len(s.text), End: len(s.text)}
}

// TextOf returns the bytes covered by span, or "" if the span belongs to
// another source.
func (s *Source) TextOf(span Span) string {
	if span.SourceID != s.id {
		return ""
	}
	return s.text[s.clamp(span.Start):s.clamp(span.End)]
}

// clamp bounds a byte offset to the source, including the end-of-file
// position.
func (s *Source) clamp(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(s.text) {
		return len(s.text)
	}
	return offset
}

// LineRange returns the half-open byte range of the given 1-based line,
// including its terminator. ok is false when the line does not exist.
func (s *Source) LineRange(line int) (start, end int, ok bool) {
	if line < 1 || line > len(s.lineStarts) {
		return 0, 0, false
	}
	start = s.lineStarts[line-1]
	if line == len(s.lineStarts) {
		return start, len(s.text), true
	}
	return start, s.lineStarts[line], true
}

// LineText returns the text of the given 1-based line without its terminator.
// A CRLF terminator is removed in full, so the returned text never ends in a
// stray carriage return that would overwrite a rendered snippet.
func (s *Source) LineText(line int) string {
	start, end, ok := s.LineRange(line)
	if !ok {
		return ""
	}
	text := s.text[start:end]
	text = trimSuffix(text, "\n")
	return trimSuffix(text, "\r")
}

// trimSuffix removes one matching suffix and otherwise returns the text
// unchanged.
func trimSuffix(text, suffix string) string {
	if len(text) >= len(suffix) && text[len(text)-len(suffix):] == suffix {
		return text[:len(text)-len(suffix)]
	}
	return text
}

// LineAt returns the 1-based line number containing offset. An offset at the
// end of the source reports the last line, which is where EOF diagnostics point.
func (s *Source) LineAt(offset int) int {
	offset = s.clamp(offset)
	// The first line start greater than offset belongs to the next line.
	return sort.Search(len(s.lineStarts), func(i int) bool { return s.lineStarts[i] > offset })
}

// Position converts a byte offset into line and column numbers using
// [DefaultTabWidth] for display columns.
func (s *Source) Position(offset int) Position {
	return s.PositionWithTabWidth(offset, DefaultTabWidth)
}

// PositionWithTabWidth converts a byte offset into line and column numbers,
// expanding tabs to the given stop width for the display column. A tabWidth
// below one is treated as one, so a tab always advances at least one cell.
//
// Every field of the result is 1-based, including the UTF-16 column. The LSP
// boundary subtracts one from Line and UTF16 when it builds a protocol
// position; keeping one convention here avoids a mixed-base struct that is easy
// to misread at a call site.
func (s *Source) PositionWithTabWidth(offset, tabWidth int) Position {
	if tabWidth < 1 {
		tabWidth = 1
	}
	offset = s.alignToRuneStart(s.clamp(offset))
	line := s.LineAt(offset)
	lineStart := s.lineStarts[line-1]
	position := Position{Line: line, Byte: offset - lineStart + 1, Rune: 1, UTF16: 1, Display: 1}
	for _, r := range s.text[lineStart:offset] {
		position.Rune++
		position.UTF16 += utf16Len(r)
		if r == '\t' {
			// Advance to the next tab stop rather than by a single cell.
			position.Display += tabWidth - (position.Display-1)%tabWidth
			continue
		}
		position.Display++
	}
	return position
}

// alignToRuneStart moves an offset back to the start of the rune containing it.
// Positions inside a multi-byte rune are a caller mistake, but reporting the
// rune's own column is more useful than counting a fragment as a character.
func (s *Source) alignToRuneStart(offset int) int {
	for offset > 0 && offset < len(s.text) && !utf8.RuneStart(s.text[offset]) {
		offset--
	}
	return offset
}

// utf16Len returns the number of UTF-16 code units needed to represent a rune.
func utf16Len(r rune) int {
	if r1, _ := utf16.EncodeRune(r); r1 == utf8.RuneError {
		return 1
	}
	return 2
}

// Position is one byte offset expressed in the units a consumer needs. Line and
// every column are 1-based and count from the start of Line.
//
// The columns differ only in what they count, and they disagree on purpose: a
// tab is one byte, one rune and one UTF-16 unit but several display cells, and
// an emoji is four bytes, one rune and two UTF-16 units.
type Position struct {
	Line int // 1-based line number
	// Byte is the 1-based column counted in bytes. Useful for slicing, wrong
	// for display once the line contains non-ASCII text.
	Byte int
	// Rune is the 1-based column counted in Unicode code points. A combining
	// mark counts as its own column because it is its own code point.
	Rune int
	// UTF16 is the 1-based column counted in UTF-16 code units. A
	// supplementary-plane character such as an emoji counts as two. Subtract
	// one at the LSP boundary, which is 0-based.
	UTF16 int
	// Display is the 1-based column counted in terminal cells with tabs
	// expanded to their stop width. Every non-tab rune counts as one cell, so
	// this does not account for double-width or zero-width characters.
	Display int
}

// Span is the canonical location type: a half-open byte range [Start, End)
// within one source. An empty span (Start == End) marks an insertion point,
// such as the end of input.
type Span struct {
	SourceID SourceID
	Start    int // byte offset, inclusive
	End      int // byte offset, exclusive
}

// NoSpan is the span of something that has no location.
var NoSpan = Span{}

// IsValid reports whether the span points at a source and is well ordered.
func (s Span) IsValid() bool {
	return s.SourceID != NoSource && s.Start >= 0 && s.End >= s.Start
}

// IsEmpty reports whether the span covers no bytes.
func (s Span) IsEmpty() bool { return s.Start == s.End }

// Len returns the number of bytes covered.
func (s Span) Len() int { return s.End - s.Start }

// Contains reports whether offset falls inside the half-open range. An empty
// span contains nothing.
func (s Span) Contains(offset int) bool { return offset >= s.Start && offset < s.End }

// Overlaps reports whether two spans of the same source share any byte.
func (s Span) Overlaps(other Span) bool {
	return s.SourceID == other.SourceID && s.Start < other.End && other.Start < s.End
}

// Union returns the smallest span covering both operands, which is how a
// composite expression derives its span from its parts. It returns [NoSpan]
// when the operands belong to different sources; a span covering two files
// would be meaningless. A union with an invalid span returns the valid one.
func (s Span) Union(other Span) Span {
	switch {
	case !s.IsValid():
		return other
	case !other.IsValid():
		return s
	case s.SourceID != other.SourceID:
		return NoSpan
	}
	return Span{SourceID: s.SourceID, Start: min(s.Start, other.Start), End: max(s.End, other.End)}
}

// Compare orders spans by source, then start, then end. It defines the
// deterministic order diagnostics are reported in.
func Compare(a, b Span) int {
	switch {
	case a.SourceID != b.SourceID:
		return cmpInt(int(a.SourceID), int(b.SourceID))
	case a.Start != b.Start:
		return cmpInt(a.Start, b.Start)
	default:
		return cmpInt(a.End, b.End)
	}
}

// cmpInt returns -1, 0, or 1 for integer ordering without subtracting its
// operands.
func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Map owns the sources of one analysis and hands out their identifiers. A
// language server analyses several documents concurrently, so the registry is
// safe for concurrent use.
type Map struct {
	mu      sync.RWMutex
	sources []*Source // index i holds the source with ID i+1
}

// NewMap returns an empty registry.
func NewMap() *Map { return &Map{} }

// Add registers text under a display filename and returns its snapshot. Adding
// the same filename twice creates a distinct snapshot; the registry does not
// deduplicate, because two snapshots of one path are exactly what an editing
// session produces.
func (m *Map) Add(name, text string) *Source {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := New(SourceID(len(m.sources)+1), name, text)
	m.sources = append(m.sources, src)
	return src
}

// Source returns the snapshot registered under id.
func (m *Map) Source(id SourceID) (*Source, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id == NoSource || int(id) > len(m.sources) {
		return nil, false
	}
	return m.sources[id-1], true
}

// Len returns the number of registered sources.
func (m *Map) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sources)
}

// Location is a position resolved for display, which no longer needs the source
// it came from.
//
// A generated executable is the reason this type exists. It carries its source
// map inside the binary and must still report main.flux:4:5 after the .flux
// file it was built from has been deleted or moved, so the filename, line, and
// column are resolved when the program is compiled rather than when it fails.
// The byte range is kept as well, for a tool that does still have the source
// and wants to show a snippet.
type Location struct {
	// SourceID preserves the originating snapshot's identity for consumers
	// that rebuild spans. It may be NoSource for display-only locations.
	SourceID SourceID
	// File is the display filename, never an absolute build-machine path.
	File string
	// Line is 1-based.
	Line int
	// Column is the 1-based display column, with tabs already expanded.
	Column int
	// Start and End are the half-open byte range within File.
	Start, End int
}

// IsValid reports whether the location identifies a place.
func (l Location) IsValid() bool { return l.File != "" && l.Line > 0 }

// String renders the location as file:line:column.
func (l Location) String() string {
	if !l.IsValid() {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

// Locate resolves a span in this source into a Location. It returns the zero
// Location for a span belonging to another source, rather than a position that
// would point into the wrong file.
func (s *Source) Locate(span Span) Location {
	if span.SourceID != s.id {
		return Location{}
	}
	position := s.Position(span.Start)
	return Location{
		SourceID: s.id,
		File:     s.name,
		Line:     position.Line,
		Column:   position.Display,
		Start:    s.clamp(span.Start),
		End:      s.clamp(span.End),
	}
}
