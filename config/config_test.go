package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfiguration(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil || !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("defaults: %v, %v", cfg, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.json"), []byte(`{"typeChecking":{"strict":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(dir)
	if err != nil || !cfg.TypeChecking.Strict || !cfg.TypeChecking.Enabled || cfg.Compiler.OptimizationLevel != 1 {
		t.Fatalf("partial config lost defaults: %v, %v", cfg, err)
	}
	cfg.TypeChecking.Enabled = false
	if err := SaveConfig(cfg, dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(dir)
	if err != nil || !reflect.DeepEqual(cfg, loaded) {
		t.Fatalf("round trip: %v, %v", loaded, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.json"), []byte(`{broken`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(dir); err == nil {
		t.Fatal("invalid config accepted")
	}
}
