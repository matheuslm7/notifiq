package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileTakesPrecedenceWhenDirectValueIsMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jwt-secret")
	if err := os.WriteFile(path, []byte("secret-from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET_FILE", path)
	t.Setenv("TEST_SECRET", "")

	if got := envOrDefault("TEST_SECRET", "fallback"); got != "secret-from-file" {
		t.Fatalf("expected file value, got %q", got)
	}
}

func TestDirectEnvTakesPrecedenceOverEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jwt-secret")
	if err := os.WriteFile(path, []byte("file-value"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET_FILE", path)
	t.Setenv("TEST_SECRET", "direct-value")

	if got := envOrDefault("TEST_SECRET", "fallback"); got != "direct-value" {
		t.Fatalf("expected direct value, got %q", got)
	}
}
