package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_InvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 0\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Run(path); err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestRun_MissingConfig(t *testing.T) {
	if err := Run(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing config")
	}
}
