package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/pranavms13/flux-lang/source"
)

// Node carries the source range of one construct. Every AST node embeds it.
//
// Participle populates a field named Pos or EndPos on any struct that has one,
// and it finds them through an embedded struct, so embedding keeps the pair out
// of twenty-odd declarations and gives every node the same accessors.
//
// The two positions form a half-open range. Pos is the first non-elided token
// of the construct, so it skips leading whitespace and comments. EndPos is the
// next raw token after the construct, which is the byte immediately following
// its last token, because the parser advances its raw cursor past that token
// before injecting the field. TestEndPositionSemantics in the parser package
// pins that behavior to the pinned Participle version; it is the kind of detail
// that a dependency upgrade changes quietly.
type Node struct {
	Pos    lexer.Position
	EndPos lexer.Position
}

// Span converts the recorded range into the canonical location type.
//
// It returns [source.NoSpan] for a node the parser never positioned, such as
// one built by hand in a test. Reporting a diagnostic at offset zero would be
// worse than reporting it without a location: it points confidently at the
// wrong construct.
func (n Node) Span(id source.SourceID) source.Span {
	if !n.HasPosition() {
		return source.NoSpan
	}
	return source.Span{SourceID: id, Start: n.Pos.Offset, End: n.EndPos.Offset}
}

// HasPosition reports whether the parser recorded a range for this node.
// Participle numbers lines from one, so a zero line means unset.
func (n Node) HasPosition() bool { return n.Pos.Line > 0 }

// Positioned is implemented by every AST node, so a diagnostic-producing
// function can accept any of them.
type Positioned interface {
	Span(id source.SourceID) source.Span
	HasPosition() bool
}
