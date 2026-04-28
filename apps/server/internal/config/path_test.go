package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathReturnsFirstExistingCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	existing := filepath.Join(tempDir, "config.example.yaml")
	writeTestFile(t, existing)

	got := ResolvePath(filepath.Join(tempDir, "missing.yaml"), existing)
	if got != existing {
		t.Fatalf("ResolvePath() = %q, want %q", got, existing)
	}
}

func TestResolvePathFallsBackToFirstCandidate(t *testing.T) {
	t.Parallel()

	first := "configs/config.example.yaml"
	got := ResolvePath(first, "../../configs/config.example.yaml")
	if got != first {
		t.Fatalf("ResolvePath() = %q, want %q", got, first)
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("app:\n  name: quantsage\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
