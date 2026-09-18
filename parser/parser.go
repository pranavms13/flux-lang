// Package parser turns Flux source into a positioned syntax tree.
package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	participlelexer "github.com/alecthomas/participle/v2/lexer"
	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/lexer"
	"github.com/pranavms13/flux-lang/source"
)

// Diagnostic codes reported by this package.
var (
	// CodeInvalidCharacter is a lexical failure: text that forms no token.
	CodeInvalidCharacter = diagnostic.Register("S_INVALID_CHARACTER",
		"the source contains text that is not a Flux token")
	// CodeUnexpectedToken is a syntax failure at a token that exists.
	CodeUnexpectedToken = diagnostic.Register("S_UNEXPECTED_TOKEN",
		"a token appears where the grammar does not allow it")
	// CodeUnexpectedEOF is a syntax failure at the end of input, which usually
	// means something was left unfinished rather than written wrongly.
	CodeUnexpectedEOF = diagnostic.Register("S_UNEXPECTED_EOF",
		"the source ends in the middle of a construct")
)

var parserInstance = participle.MustBuild[ast.Program](
	participle.Lexer(lexer.LexerRules),
	participle.Unquote("String"),
	participle.Elide("Whitespace", "SingleLineComment", "MultiLineComment"),
	participle.UseLookahead(participle.MaxLookahead),
	participle.CaseInsensitive("Keywords"),
)

// Result is what parsing one source produced.
//
// Program and Partial are deliberately separate fields. Participle returns a
// best-effort tree alongside a syntax error, which is useful to an editor that
// wants to offer completion inside a file the user is still typing, and
// disastrous to anything that executes it: the tree is missing the constructs
// the parser could not read. Keeping the usable tree in a field that is nil
// whenever parsing failed makes that impossible to get wrong by accident.
type Result struct {
	// Source is the snapshot that was parsed. Diagnostic spans refer to it.
	Source *source.Source
	// Program is the tree, and is nil when parsing failed.
	Program *ast.Program
	// Partial is the incomplete tree recovered from a failed parse. It is nil
	// when parsing succeeded, and must never be executed or compiled.
	Partial *ast.Program
	// Diagnostics describes what went wrong, in source order.
	Diagnostics []diagnostic.Diagnostic
}

// Failed reports whether parsing produced a usable tree.
func (r Result) Failed() bool { return r.Program == nil }

// Tree returns the syntax tree, whether complete or partial, for a tool that
// wants to inspect a file it cannot run. The second result says which it is.
func (r Result) Tree() (tree *ast.Program, complete bool) {
	if r.Program != nil {
		return r.Program, true
	}
	return r.Partial, false
}

// Tokens returns the token stream the parser consumed, including the comments
// and whitespace elided from the tree, or nil if parsing failed before the
// stream was recorded.
func (r Result) Tokens() []participlelexer.Token {
	if tree, _ := r.Tree(); tree != nil {
		return tree.Tokens
	}
	return nil
}

// TrailingTrivia returns the text after the last token the parser consumed. The
// token stream stops there, and on a successful parse everything beyond it is
// whitespace or comments, because a real token would have been consumed or
// reported.
func (r Result) TrailingTrivia() string {
	tree, _ := r.Tree()
	if tree == nil || r.Source == nil || !tree.HasPosition() {
		return ""
	}
	return r.Source.Text()[min(tree.EndPos.Offset, r.Source.Len()):]
}

// ParseSource parses a source snapshot, reporting failures as diagnostics
// located in it.
func ParseSource(src *source.Source) Result {
	result := Result{Source: src}
	tree, err := parserInstance.ParseString(src.Name(), src.Text())
	if err == nil {
		result.Program = tree
		return result
	}
	result.Partial = tree
	result.Diagnostics = append(result.Diagnostics, translate(err, src))
	return result
}

// Parse retains the original entry point for callers that have text and no
// source snapshot. New code should use [ParseSource] so that diagnostics carry
// the real filename instead of a placeholder.
func Parse(input string) (*ast.Program, error) {
	result := ParseSource(source.New(1, "<stdin>", input))
	if !result.Failed() {
		return result.Program, nil
	}
	return nil, errors.New(describe(result.Diagnostics[0], result.Source))
}

// describe renders a diagnostic for a caller that still expects a Go error.
// It is a stopgap: P1.6 introduces the real renderer, and this function goes
// away with the last caller of [Parse].
func describe(d diagnostic.Diagnostic, src *source.Source) string {
	position := src.Position(d.Primary.Start)
	return fmt.Sprintf("%s:%d:%d: %s", src.Name(), position.Line, position.Display, d.Message)
}

// translate converts a Participle failure into a diagnostic.
//
// It reads the typed errors rather than matching on message text. Participle
// reports a lexical failure as *lexer.Error and a grammar failure as
// *participle.UnexpectedTokenError, and both satisfy participle.Error, which
// carries the position. Message text is for humans and changes between
// releases; these types are the contract.
func translate(err error, src *source.Source) diagnostic.Diagnostic {
	var unexpected *participle.UnexpectedTokenError
	if errors.As(err, &unexpected) {
		return fromUnexpectedToken(unexpected, src)
	}

	var lexical *participlelexer.Error
	if errors.As(err, &lexical) {
		// Participle's message quotes the whole unmatched remainder of the
		// input, which would underline text that is mostly fine. Report the
		// first character instead, which is the one that forms no token.
		span := spanOfRune(src, lexical.Position().Offset)
		return diagnostic.Error(CodeInvalidCharacter, span,
			"unexpected character %q", src.TextOf(span))
	}

	var positioned participle.Error
	if errors.As(err, &positioned) {
		return diagnostic.Error(CodeUnexpectedToken,
			src.Span(positioned.Position().Offset, positioned.Position().Offset),
			"%s", positioned.Message())
	}

	// Not a parse failure at all: an I/O error from the reader, or a defect.
	return diagnostic.Internal(src.Whole(), "parser reported an unrecognized failure: %v", err)
}

// fromUnexpectedToken locates an unexpected token or reports an unfinished
// construct at the end of the source.
func fromUnexpectedToken(err *participle.UnexpectedTokenError, src *source.Source) diagnostic.Diagnostic {
	start := err.Unexpected.Pos.Offset
	span := src.Span(start, start+len(err.Unexpected.Value))
	if err.Unexpected.EOF() {
		// There is no token to underline, so the span is the empty position at
		// the end of input, and the message says what is missing rather than
		// what was found.
		return diagnostic.Error(CodeUnexpectedEOF, src.EOF(),
			"unexpected end of input%s", expectation(err)).
			WithNote("a construct started earlier in the file was never finished")
	}
	return diagnostic.Error(CodeUnexpectedToken, span,
		"unexpected %s%s", describeToken(err.Unexpected), expectation(err))
}

// expectation extracts what the grammar allowed here. Participle keeps the
// expected set in an unexported field and only renders it into Message, so the
// text is recovered from there rather than rebuilt.
func expectation(err *participle.UnexpectedTokenError) string {
	message := err.Message()
	open := strings.Index(message, " (expected ")
	if open < 0 || !strings.HasSuffix(message, ")") {
		return ""
	}
	return ", expected " + message[open+len(" (expected "):len(message)-1]
}

// describeToken quotes a token value for a diagnostic, falling back to a
// generic label for an empty value.
func describeToken(token participlelexer.Token) string {
	if token.Value == "" {
		return "token"
	}
	return fmt.Sprintf("%q", token.Value)
}

// spanOfRune returns the span of the single character at an offset, so that a
// lexical failure underlines one character rather than the rest of the line.
func spanOfRune(src *source.Source, start int) source.Span {
	text := src.Text()
	if start >= len(text) {
		return src.EOF()
	}
	width := 1
	for start+width < len(text) && !isRuneStart(text[start+width]) {
		width++
	}
	return src.Span(start, start+width)
}

// isRuneStart reports whether a byte is not a UTF-8 continuation byte.
func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
