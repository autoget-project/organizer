package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile_MissingFileIgnored(t *testing.T) {
	t.Parallel()

	LoadEnvFile(t, filepath.Join(t.TempDir(), "absent.env"))
}

func TestLoadEnvFile_LoadsKeysAndKeepsExisting(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel, so this case stays serial.
	path := filepath.Join(t.TempDir(), ".env.e2e")
	content := "# provider keys\n" +
		"XAI_API_KEY=xai-fake\n" +
		"\n" +
		"MODEL=gemini:gemini-3.5-flash-lite\n" +
		"BAD_LINE_WITHOUT_EQUALS\n" +
		"  SPACED_KEY  =  spaced value  \n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("MODEL", "preset-from-shell")

	LoadEnvFile(t, path)

	if got := os.Getenv("XAI_API_KEY"); got != "xai-fake" {
		t.Errorf("new key must be loaded, got %q", got)
	}
	if got := os.Getenv("MODEL"); got != "preset-from-shell" {
		t.Errorf("already-set variable must keep precedence, got %q", got)
	}
	if got := os.Getenv("SPACED_KEY"); got != "spaced value" {
		t.Errorf("keys/values must be trimmed, got %q", got)
	}
}
