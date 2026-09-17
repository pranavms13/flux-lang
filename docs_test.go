package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/diagnostic"

	// Imported for their code registrations. The reference is generated from
	// the registry, so a stage whose codes are not registered here would be
	// missing from the document without anything noticing.
	_ "github.com/pranavms13/flux-lang/fault"
	_ "github.com/pranavms13/flux-lang/parser"
	_ "github.com/pranavms13/flux-lang/types"
)

const diagnosticReferencePath = "docs/DIAGNOSTICS.md"

// TestDiagnosticReferenceIsCurrent keeps the published code reference equal to
// the codes that exist. A reference maintained by hand beside the code is a
// reference that is wrong within a month, and a user who looks up a code and
// does not find it learns to stop looking.
//
// Run with FLUX_UPDATE_DOCS=1 to rewrite the file.
func TestDiagnosticReferenceIsCurrent(t *testing.T) {
	want := diagnosticReference()

	if os.Getenv("FLUX_UPDATE_DOCS") != "" {
		if err := os.WriteFile(diagnosticReferencePath, []byte(want), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("rewrote %s", diagnosticReferencePath)
		return
	}

	got, err := os.ReadFile(diagnosticReferencePath)
	if err != nil {
		t.Fatalf("%v\nrun FLUX_UPDATE_DOCS=1 go test -run TestDiagnosticReferenceIsCurrent .", err)
	}
	if string(got) != want {
		t.Errorf("%s is out of date; run FLUX_UPDATE_DOCS=1 go test -run TestDiagnosticReferenceIsCurrent .\n%s",
			diagnosticReferencePath, firstDifference(string(got), want))
	}
}

// TestEveryRegisteredCodeHasAGroup checks the invariant the reference relies
// on: a code with no group would be silently left out of every section.
func TestEveryRegisteredCodeHasAGroup(t *testing.T) {
	codes := diagnostic.Registered()
	if len(codes) < 20 {
		t.Fatalf("only %d codes are registered; the stages' inits did not run", len(codes))
	}
	for _, code := range codes {
		if _, ok := code.Group(); !ok {
			t.Errorf("%s has no reserved group", code)
		}
		if description, ok := diagnostic.Describe(code); !ok || description == "" {
			t.Errorf("%s has no description", code)
		}
	}
}

func diagnosticReference() string {
	byGroup := map[diagnostic.Group][]diagnostic.Code{}
	for _, code := range diagnostic.Registered() {
		group, ok := code.Group()
		if !ok {
			continue
		}
		byGroup[group] = append(byGroup[group], code)
	}

	var out strings.Builder
	out.WriteString(`# Diagnostic codes

Every diagnostic Flux reports carries a code. The code is the stable part: the
wording of a message may improve, but a code keeps its meaning, so it is what to
search for, assert on in a test, or filter by in an editor.

A code's prefix names the stage that reports it, which usually tells you what
kind of mistake it is before you look it up.

This file is generated from the code registry. To update it, run:

`)
	out.WriteString("```sh\nFLUX_UPDATE_DOCS=1 go test -run TestDiagnosticReferenceIsCurrent .\n```\n")

	for _, group := range diagnostic.Groups {
		codes := byGroup[group]
		sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
		fmt.Fprintf(&out, "\n## %s — `%s_`\n\n%s\n\n", group.Name(), group, group.Description())
		out.WriteString("| Code | Meaning |\n| --- | --- |\n")
		for _, code := range codes {
			description, _ := diagnostic.Describe(code)
			fmt.Fprintf(&out, "| `%s` | %s |\n", code, description)
		}
	}
	return out.String()
}

func firstDifference(got, want string) string {
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		g, w := at(gotLines, i), at(wantLines, i)
		if g != w {
			return fmt.Sprintf("first difference at line %d:\n  file:      %q\n  generated: %q", i+1, g, w)
		}
	}
	return ""
}

func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "<end of file>"
}
