// Package testutil contains helpers for integration tests.
package testutil

import (
	"os"
	"testing"
)

// CaptureOutput uses a file so large outputs cannot fill an unread pipe.
// Tests using this helper must not run in parallel because os.Stdout is global.
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = old; f.Close() }()
	fn()
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
