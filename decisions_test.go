package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/internal/conformance"
)

const (
	decisionsDir  = "docs/decisions"
	migrationPath = "docs/MIGRATION.md"
)

// decision is one record under docs/decisions.
type decision struct {
	// Path is the record's file.
	Path string
	// Rules are the specification rules the record decides.
	Rules []string
	// Status is "accepted" or "superseded".
	Status string
	// Phase is "implemented", or the phase number that carries the change.
	Phase string
	// Slice names the slice of docs/PLAN.md that implements it.
	Slice string
	// Migration is "none", or the anchor in docs/MIGRATION.md that explains
	// the change to an existing program.
	Migration string
}

// decisionField matches one header line of a record: "- Rules: `A`, `B`".
var decisionField = regexp.MustCompile(`^- (Status|Rules|Phase|Slice|Migration): (.+)$`)

// migrationLink matches a link into the migration notes, from which only the
// anchor matters.
var migrationLink = regexp.MustCompile(`\(\.\./MIGRATION\.md#([a-z0-9-]+)\)`)

// readDecisions loads every record, failing on one whose header is incomplete:
// a record that does not say which rules it decides is a note, not a decision.
func readDecisions(t *testing.T) []decision {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(decisionsDir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	var decisions []decision
	for _, path := range paths {
		if filepath.Base(path) == "README.md" {
			continue
		}
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		record := decision{Path: filepath.ToSlash(path)}
		for _, line := range strings.Split(string(text), "\n") {
			match := decisionField.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			switch value := strings.TrimSpace(match[2]); match[1] {
			case "Status":
				record.Status = value
			case "Phase":
				record.Phase = value
			case "Slice":
				record.Slice = value
			case "Migration":
				record.Migration = value
			case "Rules":
				record.Rules = backticked(value)
			}
		}
		decisions = append(decisions, record)
	}
	if len(decisions) == 0 {
		t.Fatalf("%s holds no decision records", decisionsDir)
	}
	return decisions
}

// backticked extracts the identifiers in a comma-separated backticked list.
func backticked(value string) []string {
	var found []string
	for _, match := range regexp.MustCompile("`([A-Z][A-Z0-9-]*)`").FindAllStringSubmatch(value, -1) {
		found = append(found, match[1])
	}
	return found
}

// TestDecisionsAreLinked checks that a decision, the rules it decides and the
// migration it promises stay attached to each other.
//
// The three drift apart in a predictable order: a rule is renamed and its
// record still names the old one; a decision is written and its migration
// section never is; a planned rule is added with no record of why. Each of
// those is one assertion here.
func TestDecisionsAreLinked(t *testing.T) {
	rules := map[string]specRule{}
	for _, rule := range readSpecRules(t) {
		rules[rule.ID] = rule
	}
	anchors := migrationAnchors(t)
	decisions := readDecisions(t)

	decided := map[string][]decision{}
	for _, record := range decisions {
		t.Run(filepath.Base(record.Path), func(t *testing.T) {
			switch {
			case record.Status == "":
				t.Error("no status")
			case record.Phase == "":
				t.Error("no phase; say \"implemented\" or the phase number that carries it")
			case record.Slice == "":
				t.Error("no slice; say which slice of docs/PLAN.md implements it, or \"none\"")
			case record.Migration == "":
				t.Error("no migration; say \"none\" or link the section that explains the change")
			case len(record.Rules) == 0:
				t.Error("names no specification rule")
			}
			// A record usually names a mix: the rules it changes, and the
			// rules it deliberately preserves alongside them. So a named rule
			// must be either already implemented or planned for this record's
			// phase, and a record that claims a phase must change something.
			changes := false
			for _, id := range record.Rules {
				rule, ok := rules[id]
				if !ok {
					t.Errorf("names %s, which %s does not declare", id, specPath)
					continue
				}
				decided[id] = append(decided[id], record)
				if rule.Status == conformance.StatusImplemented {
					continue
				}
				changes = true
				if want := phaseOf(rule); want != record.Phase {
					t.Errorf("says phase %q, but %s plans %s for phase %q",
						record.Phase, specPath, id, want)
				}
			}
			if changes == (record.Phase == "implemented") {
				t.Errorf("says phase %q, which does not match the rules it names: "+
					"a record that changes nothing is \"implemented\", and one that "+
					"changes something names the phase that does it", record.Phase)
			}
			if anchor := migrationLink.FindStringSubmatch(record.Migration); anchor != nil {
				if !anchors[anchor[1]] {
					t.Errorf("points at %s#%s, which has no such section", migrationPath, anchor[1])
				}
			} else if record.Migration != "none" && !strings.HasPrefix(record.Migration, "none") {
				t.Errorf("migration is %q; it must be \"none\" or a link into %s",
					record.Migration, migrationPath)
			}
		})
	}

	// A planned rule with no record is a change nobody wrote the reason for,
	// which is exactly the kind that gets reversed by the next implementer.
	var undecided []string
	for id, rule := range rules {
		if rule.Status == conformance.StatusPlanned && len(decided[id]) == 0 {
			undecided = append(undecided, id)
		}
	}
	sort.Strings(undecided)
	if len(undecided) > 0 {
		t.Errorf("no decision record explains %s; every planned rule needs one in %s",
			strings.Join(undecided, ", "), decisionsDir)
	}
}

// phaseOf renders a rule's status the way a decision record writes its phase.
func phaseOf(rule specRule) string {
	if rule.Status == conformance.StatusImplemented {
		return "implemented"
	}
	return strings.TrimPrefix(rule.Milestone, "phase-")
}

// migrationAnchors returns the heading anchors docs/MIGRATION.md defines.
func migrationAnchors(t *testing.T) map[string]bool {
	t.Helper()
	text, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	anchors := map[string]bool{}
	for _, line := range strings.Split(string(text), "\n") {
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			anchors[slug(heading)] = true
		}
	}
	if len(anchors) == 0 {
		t.Fatalf("%s has no sections", migrationPath)
	}
	return anchors
}

// slug converts a Markdown heading into the anchor a link uses.
func slug(heading string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			out.WriteRune(r)
		case r == ' ':
			out.WriteRune('-')
		}
	}
	return out.String()
}

// TestDecisionIndexIsComplete checks that the index lists every record. An
// index that is missing one is worse than no index: it reads as the full set.
func TestDecisionIndexIsComplete(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(decisionsDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range readDecisions(t) {
		name := filepath.Base(record.Path)
		if !strings.Contains(string(index), "("+name+")") {
			t.Errorf("%s/README.md does not link %s", decisionsDir, name)
		}
	}
}
