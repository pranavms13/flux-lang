// Package conformance loads the executable form of the language
// specification.
//
// Every normative rule in docs/SPEC.md is followed by at least one fixture
// under testdata/conformance/. A fixture is an ordinary Flux file whose header
// comments declare what the rule requires: which rule it belongs to, what the
// program is expected to do in each type-checking mode, and, for a failure,
// which diagnostic code is reported and where. The header is written in
// comments so the fixture is still a program the real parser accepts; nothing
// in it is a second, hand-maintained copy of the source.
//
// A fixture is either implemented or planned. An implemented fixture states
// the behavior Flux has today, and the harness asserts it. A planned fixture
// states behavior a later phase will introduce, and records both that
// requirement and the behavior Flux has in the meantime; the harness asserts
// the present-day behavior and fails when it starts to match the requirement,
// so the fixture must be promoted rather than quietly passing. Neither kind is
// ever skipped.
package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pranavms13/flux-lang/internal/fixtures"
)

// Directive is the prefix that marks a header line. It is a comment to the
// lexer, and the doubled slash plus bang distinguishes it from an ordinary
// explanatory comment inside the same file.
const Directive = "//!"

// Status says whether a fixture describes what Flux does or what it is
// specified to do later.
type Status string

const (
	// StatusImplemented marks a rule the current implementation satisfies.
	StatusImplemented Status = "implemented"
	// StatusPlanned marks a rule a later phase introduces. The fixture also
	// records today's behavior so it still asserts something.
	StatusPlanned Status = "planned"
)

// Kind is the shape of one expected result.
type Kind string

const (
	// KindOutput means the program is accepted and runs to completion.
	KindOutput Kind = "output"
	// KindStaticError means the program is rejected before it runs, by the
	// parser or the checker.
	KindStaticError Kind = "static-error"
	// KindRuntimeError means the program is accepted and then fails while
	// running.
	KindRuntimeError Kind = "runtime-error"
)

// IsError reports whether the outcome is a reported failure rather than a
// completed run.
func (k Kind) IsError() bool { return k == KindStaticError || k == KindRuntimeError }

// Outcome is what one mode is expected to produce.
//
// Code and Span are required for a failure and absent for a completed run. The
// span is what makes a diagnostic fixture worth having: a code alone does not
// say whether the message underlines the mistake or the construct after it.
type Outcome struct {
	Kind Kind
	// Code is the diagnostic code, such as "T_ANNOTATION_MISMATCH".
	Code string
	// Span is the range the diagnostic underlines.
	Span Span
}

// Warning is a diagnostic a mode reports without rejecting the program.
type Warning struct {
	Code string
	Span Span
}

// String renders the warning the way a fixture header writes it.
func (w Warning) String() string { return fmt.Sprintf("%s at %s", w.Code, w.Span) }

// Span is a source range written as line:column..line:column, in 1-based lines
// and display columns — the same pair the renderer prints.
type Span struct {
	StartLine, StartColumn int
	EndLine, EndColumn     int
}

// String renders the span the way a fixture header writes it.
func (s Span) String() string {
	return fmt.Sprintf("%d:%d..%d:%d", s.StartLine, s.StartColumn, s.EndLine, s.EndColumn)
}

// IsZero reports whether no span was declared.
func (s Span) IsZero() bool { return s == Span{} }

// String renders the outcome the way a fixture header writes it.
func (o Outcome) String() string {
	if !o.Kind.IsError() {
		return string(o.Kind)
	}
	return fmt.Sprintf("%s %s at %s", o.Kind, o.Code, o.Span)
}

// Fixture is one conformance case.
type Fixture struct {
	// Path is the fixture file, relative to the repository root.
	Path string
	// Rule is the identifier of the docs/SPEC.md rule this case pins.
	Rule string
	// About says what the case proves, in one line.
	About string
	// Status says whether the rule is implemented or planned.
	Status Status
	// Milestone names the phase that implements a planned rule, such as
	// "phase-3". It is empty for an implemented rule.
	Milestone string
	// Specified is the outcome the rule requires. It is set only for a planned
	// rule, where it differs from every entry in Outcomes.
	Specified Outcome
	// SpecifiedStdout is the output the rule requires, for a planned rule
	// whose Specified outcome completes.
	SpecifiedStdout string
	// Outcomes is the present-day result in every mode of [fixtures.Modes].
	Outcomes map[fixtures.Mode]Outcome
	// Warnings is the diagnostic a mode is expected to report without failing
	// the program, for the modes that declare one. It is what makes
	// MOD-SEVERITY-ONLY checkable: a rule that is an error in strict mode and a
	// warning elsewhere must keep the same code and the same span.
	Warnings map[fixtures.Mode]Warning
	// Stdout is the output expected wherever an outcome completes. A program
	// that completes in several modes prints the same thing in all of them,
	// because type checking decides whether a program runs, not what it
	// prints.
	Stdout string
	// stdoutDeclared and specifiedStdoutDeclared record that the directive was
	// written, which is not the same as the output being non-empty: a program
	// that prints nothing is a result worth declaring, and one whose stdout was
	// forgotten is a fixture that checks less than it appears to.
	stdoutDeclared          bool
	specifiedStdoutDeclared bool
	// Source is the fixture's full text, header comments included, so that the
	// declared spans refer to the same bytes the parser sees.
	Source string
}

// Name returns the fixture's path below the conformance root, which reads
// better as a subtest name than the full path.
func (f Fixture) Name() string {
	if index := strings.Index(f.Path, Root+"/"); index >= 0 {
		return f.Path[index+len(Root)+1:]
	}
	return f.Path
}

// Outcome returns the present-day result in one mode.
func (f Fixture) Outcome(mode fixtures.Mode) Outcome { return f.Outcomes[mode] }

// Root is the directory holding the conformance suite, relative to the
// repository root.
const Root = "testdata/conformance"

// Load reads every fixture under dir, in path order.
//
// It returns an error rather than a partial suite: a header that does not
// parse is a fixture that asserts nothing, and silently dropping it is how a
// specification stops being executable.
func Load(dir string) ([]Fixture, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".flux") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	loaded := make([]Fixture, 0, len(paths))
	for _, path := range paths {
		text, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fixture, err := Parse(filepath.ToSlash(path), string(text))
		if err != nil {
			return nil, err
		}
		loaded = append(loaded, fixture)
	}
	return loaded, nil
}

// Parse reads a fixture's header and returns the declared case.
func Parse(path, text string) (Fixture, error) {
	fixture := Fixture{
		Path: path, Source: text,
		Outcomes: map[fixtures.Mode]Outcome{},
		Warnings: map[fixtures.Mode]Warning{},
	}
	for number, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, Directive) {
			// The header is the run of directives at the top of the file.
			// Anything else ends it, so a later comment cannot change what a
			// fixture asserts from somewhere a reader would not look.
			if trimmed == "" {
				continue
			}
			break
		}
		key, value, ok := split(strings.TrimSpace(strings.TrimPrefix(trimmed, Directive)))
		if !ok {
			return fixture, fmt.Errorf("%s:%d: directive needs a %q", path, number+1, ":")
		}
		if err := fixture.apply(key, value); err != nil {
			return fixture, fmt.Errorf("%s:%d: %w", path, number+1, err)
		}
	}
	if err := fixture.Validate(); err != nil {
		return fixture, err
	}
	return fixture, nil
}

// apply records one directive.
func (f *Fixture) apply(key, value string) error {
	switch key {
	case "rule":
		f.Rule = value
	case "about":
		f.About = value
	case "status":
		switch Status(value) {
		case StatusImplemented, StatusPlanned:
			f.Status = Status(value)
		default:
			return fmt.Errorf("status is %q, want %q or %q", value, StatusImplemented, StatusPlanned)
		}
	case "milestone":
		f.Milestone = value
	case "specified":
		outcome, err := ParseOutcome(value)
		if err != nil {
			return err
		}
		f.Specified = outcome
	case "specified-stdout":
		text, err := strconv.Unquote(value)
		if err != nil {
			return fmt.Errorf("specified-stdout must be a quoted string: %w", err)
		}
		f.SpecifiedStdout, f.specifiedStdoutDeclared = text, true
	case "stdout":
		text, err := strconv.Unquote(value)
		if err != nil {
			return fmt.Errorf("stdout must be a quoted string: %w", err)
		}
		f.Stdout, f.stdoutDeclared = text, true
	case "all":
		// Shorthand for a program that behaves the same way everywhere. A
		// later mode directive overrides it, which is how a mode-dependent
		// fixture states only the mode that differs.
		outcome, err := ParseOutcome(value)
		if err != nil {
			return err
		}
		for _, mode := range fixtures.Modes {
			f.Outcomes[mode] = outcome
		}
	default:
		if name, isWarning := strings.CutSuffix(key, "-warning"); isWarning {
			mode := fixtures.Mode(name)
			if !known(mode) {
				return fmt.Errorf("unknown mode %q in directive %q", name, key)
			}
			warning, err := ParseWarning(value)
			if err != nil {
				return err
			}
			f.Warnings[mode] = warning
			return nil
		}
		mode := fixtures.Mode(key)
		if !known(mode) {
			return fmt.Errorf("unknown directive %q", key)
		}
		outcome, err := ParseOutcome(value)
		if err != nil {
			return err
		}
		f.Outcomes[mode] = outcome
	}
	return nil
}

// ParseOutcome reads an outcome written as "output", or as
// "static-error CODE at line:col..line:col".
func ParseOutcome(value string) (Outcome, error) {
	kind, rest, hasRest := split2(value, " ")
	outcome := Outcome{Kind: Kind(kind)}
	switch outcome.Kind {
	case KindOutput:
		if hasRest {
			return outcome, fmt.Errorf("%q takes no code or span", KindOutput)
		}
		return outcome, nil
	case KindStaticError, KindRuntimeError:
	default:
		return outcome, fmt.Errorf("unknown outcome %q", kind)
	}
	if !hasRest {
		return outcome, fmt.Errorf("%q needs a diagnostic code and a span", kind)
	}
	code, location, ok := split2(rest, " at ")
	if !ok {
		return outcome, fmt.Errorf("%q needs \" at line:col..line:col\"", rest)
	}
	span, err := ParseSpan(location)
	if err != nil {
		return outcome, err
	}
	outcome.Code, outcome.Span = strings.TrimSpace(code), span
	return outcome, nil
}

// ParseWarning reads a warning written as "CODE at line:col..line:col".
func ParseWarning(value string) (Warning, error) {
	code, location, ok := split2(value, " at ")
	if !ok {
		return Warning{}, fmt.Errorf("warning %q needs \" at line:col..line:col\"", value)
	}
	span, err := ParseSpan(location)
	if err != nil {
		return Warning{}, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return Warning{}, fmt.Errorf("warning %q must name a diagnostic code", value)
	}
	return Warning{Code: code, Span: span}, nil
}

// ParseSpan reads a span written as line:col..line:col.
func ParseSpan(text string) (Span, error) {
	start, end, ok := split2(strings.TrimSpace(text), "..")
	if !ok {
		return Span{}, fmt.Errorf("span %q must be line:col..line:col", text)
	}
	startLine, startColumn, err := parsePosition(start)
	if err != nil {
		return Span{}, err
	}
	endLine, endColumn, err := parsePosition(end)
	if err != nil {
		return Span{}, err
	}
	return Span{
		StartLine: startLine, StartColumn: startColumn,
		EndLine: endLine, EndColumn: endColumn,
	}, nil
}

// parsePosition reads one 1-based line:column pair.
func parsePosition(text string) (line, column int, err error) {
	rawLine, rawColumn, ok := split2(text, ":")
	if !ok {
		return 0, 0, fmt.Errorf("position %q must be line:column", text)
	}
	if line, err = strconv.Atoi(rawLine); err != nil || line < 1 {
		return 0, 0, fmt.Errorf("position %q has no 1-based line", text)
	}
	if column, err = strconv.Atoi(rawColumn); err != nil || column < 1 {
		return 0, 0, fmt.Errorf("position %q has no 1-based column", text)
	}
	return line, column, nil
}

// Validate reports whether the declaration is coherent.
//
// It catches the mistakes that would make a fixture pass without proving
// anything: a missing outcome for a mode, an expected output with nothing to
// compare it against, or a planned rule whose requirement is already what the
// fixture says happens today.
func (f Fixture) Validate() error {
	switch {
	case f.Rule == "":
		return fmt.Errorf("fixture %q: must name the docs/SPEC.md rule it pins", f.Path)
	case f.About == "":
		return fmt.Errorf("fixture %q: must say what it proves", f.Path)
	case f.Status == "":
		return fmt.Errorf("fixture %q: must declare a status", f.Path)
	}

	completes := false
	for _, mode := range fixtures.Modes {
		outcome, declared := f.Outcomes[mode]
		if !declared {
			return fmt.Errorf("fixture %q: no outcome declared for mode %q", f.Path, mode)
		}
		if err := outcome.validate(f.Path, string(mode)); err != nil {
			return err
		}
		if outcome.Kind == KindOutput {
			completes = true
		}
	}
	if completes != f.stdoutDeclared {
		return fmt.Errorf("fixture %q: a mode that completes must declare stdout, and only then", f.Path)
	}

	if f.Status == StatusImplemented {
		switch {
		case f.Milestone != "":
			return fmt.Errorf("fixture %q: an implemented rule has no milestone", f.Path)
		case f.Specified != (Outcome{}):
			return fmt.Errorf("fixture %q: an implemented rule's outcomes are the specification", f.Path)
		case f.specifiedStdoutDeclared:
			return fmt.Errorf("fixture %q: an implemented rule's stdout is the specification", f.Path)
		}
		return nil
	}

	if f.Milestone == "" {
		return fmt.Errorf("fixture %q: a planned rule must name the phase that implements it", f.Path)
	}
	if err := f.Specified.validate(f.Path, "specified"); err != nil {
		return err
	}
	if (f.Specified.Kind == KindOutput) != f.specifiedStdoutDeclared {
		return fmt.Errorf("fixture %q: a specified outcome that completes must declare specified-stdout, and only then", f.Path)
	}
	// A planned rule that already agrees with today's behavior in every mode is
	// an implemented rule whose fixture nobody updated. Agreeing in some modes
	// is ordinary: a rule such as "in every mode" is often already satisfied by
	// the strictest one.
	if f.SatisfiesSpecification() {
		return fmt.Errorf("fixture %q: every mode already matches the specified outcome; "+
			"mark the rule implemented", f.Path)
	}
	return nil
}

// SatisfiesSpecification reports whether the recorded present-day behavior
// already meets the rule in every mode, which is what "implemented" means. A
// well-formed planned fixture returns false.
func (f Fixture) SatisfiesSpecification() bool {
	for _, mode := range fixtures.Modes {
		if f.Outcomes[mode] != f.Specified {
			return false
		}
	}
	return f.Specified.Kind.IsError() || f.Stdout == f.SpecifiedStdout
}

// validate checks that an outcome carries exactly the fields its kind needs.
func (o Outcome) validate(path, where string) error {
	switch {
	case o.Kind == "":
		return fmt.Errorf("fixture %q: %s declares no outcome", path, where)
	case !o.Kind.IsError():
		if o.Code != "" || !o.Span.IsZero() {
			return fmt.Errorf("fixture %q: %s completes, so it has no code or span", path, where)
		}
	case o.Code == "":
		return fmt.Errorf("fixture %q: %s must name the diagnostic code", path, where)
	case o.Span.IsZero():
		return fmt.Errorf("fixture %q: %s must name the span the diagnostic underlines", path, where)
	}
	return nil
}

// Implemented returns the fixtures whose rules the current implementation
// satisfies.
func Implemented(all []Fixture) []Fixture { return withStatus(all, StatusImplemented) }

// Planned returns the fixtures whose rules a later phase introduces.
func Planned(all []Fixture) []Fixture { return withStatus(all, StatusPlanned) }

func withStatus(all []Fixture, status Status) []Fixture {
	var selected []Fixture
	for _, fixture := range all {
		if fixture.Status == status {
			selected = append(selected, fixture)
		}
	}
	return selected
}

// Codes returns every diagnostic code the suite expects to see, in sorted
// order. A planned fixture contributes the codes it reports today as well as
// the one its rule requires, because both are asserted somewhere.
func Codes(all []Fixture) []string {
	seen := map[string]bool{}
	for _, fixture := range all {
		for _, outcome := range fixture.Outcomes {
			if outcome.Code != "" {
				seen[outcome.Code] = true
			}
		}
		if fixture.Specified.Code != "" {
			seen[fixture.Specified.Code] = true
		}
	}
	codes := make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// known reports whether a mode is one the fixture manifest defines.
func known(mode fixtures.Mode) bool {
	for _, declared := range fixtures.Modes {
		if declared == mode {
			return true
		}
	}
	return false
}

// split separates a directive's key from its value at the first colon.
func split(line string) (key, value string, ok bool) {
	key, value, ok = split2(line, ":")
	return strings.TrimSpace(key), strings.TrimSpace(value), ok
}

// split2 is strings.Cut with the separator's presence reported, kept as one
// helper so the parsing above reads the same way everywhere.
func split2(text, separator string) (before, after string, found bool) {
	before, after, found = strings.Cut(text, separator)
	return before, strings.TrimSpace(after), found
}
