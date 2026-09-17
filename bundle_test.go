package main

import (
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/pranavms13/flux-lang"

// TestBuildBundleIsClosed is what makes the bundle safe to rely on. A generated
// executable is built from these packages alone, with the module proxy off, so
// an import that is not in the bundle turns into a build failure on a user's
// machine — long after the change that caused it. This test moves that failure
// here.
func TestBuildBundleIsClosed(t *testing.T) {
	bundled := map[string]bool{}
	for _, pkg := range bundledPackages {
		bundled[pkg] = true
	}

	var checked int
	for _, pkg := range bundledPackages {
		entries, err := bundleFS.ReadDir(pkg)
		if err != nil {
			t.Fatalf("bundle package %s: %v", pkg, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			contents, err := bundleFS.ReadFile(path.Join(pkg, name))
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), name, contents, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("%s/%s: %v", pkg, name, err)
			}
			checked++
			for _, imported := range file.Imports {
				assertImportable(t, pkg+"/"+name, strings.Trim(imported.Path.Value, `"`), bundled)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no bundled sources were checked; the embed patterns match nothing")
	}
}

func assertImportable(t *testing.T, file, imported string, bundled map[string]bool) {
	t.Helper()
	switch {
	case strings.HasPrefix(imported, modulePath+"/"):
		pkg := strings.TrimPrefix(imported, modulePath+"/")
		if !bundled[pkg] {
			t.Errorf("%s imports %s, which is not in the bundle; either add it to "+
				"bundledPackages or keep it out of the executable's dependencies", file, pkg)
		}
	case strings.Contains(strings.Split(imported, "/")[0], "."):
		// A dot in the first element means a hosted module. The bundle has no
		// go.sum and builds with the proxy disabled, so it cannot resolve one.
		t.Errorf("%s imports the external module %s; the bundle can only use the standard library", file, imported)
	}
}

// TestBuildBundleExcludesTests checks that what is written to disk is what the
// bundle claims to contain. Test files would drag in testing-only imports that
// the bundle does not have.
func TestBuildBundleExcludesTests(t *testing.T) {
	dir := t.TempDir()
	if err := writeBundle(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Errorf("the bundle has no go.mod: %v", err)
	}
	var written int
	err := filepath.WalkDir(dir, func(p string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") {
			t.Errorf("%s was written into the bundle", p)
		}
		if strings.HasSuffix(name, ".go") {
			written++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if written < len(bundledPackages) {
		t.Errorf("wrote %d sources for %d packages", written, len(bundledPackages))
	}

	// The checkout has test files in these packages, so the exclusion above is
	// actually exercising something.
	for _, pkg := range []string{"source", "diagnostic"} {
		entries, err := bundleFS.ReadDir(pkg)
		if err != nil {
			t.Fatal(err)
		}
		var hasTests bool
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), "_test.go") {
				hasTests = true
			}
		}
		if !hasTests {
			t.Errorf("%s has no test files, so the exclusion is untested", pkg)
		}
	}
}
