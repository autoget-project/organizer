package config

import (
	"os"
	"path/filepath"
	"testing"

	"organizer/internal/model"
)

func TestStartupCheckMissingEnv(t *testing.T) {
	cfg := &Config{}
	if err := StartupCheck(cfg); err == nil {
		t.Errorf("expected error for empty config, got nil")
	}

	cfg.DownloadCompletedDir = "/tmp/download"
	if err := StartupCheck(cfg); err == nil {
		t.Errorf("expected error when TARGET_DIR missing")
	}
}

func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		xaiKey    string
		geminiKey string
		expected  string
		wantErr   bool
	}{
		{
			name:      "neither key provided",
			model:     "grok-beta",
			xaiKey:    "",
			geminiKey: "",
			wantErr:   true,
		},
		{
			name:      "xai prefix with key",
			model:     "xai:grok-2",
			xaiKey:    "xai-123",
			geminiKey: "",
			expected:  "grok",
			wantErr:   false,
		},
		{
			name:      "xai prefix without key",
			model:     "xai:grok-2",
			xaiKey:    "",
			geminiKey: "gemini-123",
			wantErr:   true,
		},
		{
			name:      "gemini prefix with key",
			model:     "gemini:gemini-1.5-flash",
			xaiKey:    "",
			geminiKey: "gemini-123",
			expected:  "gemini",
			wantErr:   false,
		},
		{
			name:      "gemini prefix without key",
			model:     "gemini:gemini-1.5-flash",
			xaiKey:    "xai-123",
			geminiKey: "",
			wantErr:   true,
		},
		{
			name:      "both keys present and xai prefix",
			model:     "xai:grok-beta",
			xaiKey:    "xai-123",
			geminiKey: "gemini-123",
			expected:  "grok",
			wantErr:   false,
		},
		{
			name:      "both keys present and gemini prefix",
			model:     "gemini:gemini-1.5-pro",
			xaiKey:    "xai-123",
			geminiKey: "gemini-123",
			expected:  "gemini",
			wantErr:   false,
		},
		{
			name:      "both keys present without prefix or keyword",
			model:     "custom-model",
			xaiKey:    "xai-123",
			geminiKey: "gemini-123",
			wantErr:   true,
		},
		{
			name:      "single xai key fallback",
			model:     "custom-model",
			xaiKey:    "xai-123",
			geminiKey: "",
			expected:  "grok",
			wantErr:   false,
		},
		{
			name:      "single gemini key fallback",
			model:     "custom-model",
			xaiKey:    "",
			geminiKey: "gemini-123",
			expected:  "gemini",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProvider(tt.model, tt.xaiKey, tt.geminiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ResolveProvider() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStartupCheckDirectories(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	downloadDir := filepath.Join(tempDir, "download")

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		DownloadCompletedDir: downloadDir,
		TargetDir:            targetDir,
		JavActorFile:         filepath.Join(tempDir, "actor.json"),
		FlareSolverrURL:      "http://localhost:8191",
		MetadataMCP:          "http://localhost:8000/sse",
		Model:                "xai:grok-2",
		XaiAPIKey:            "valid-key",
	}

	// targetDir does not exist yet -> should fail
	if err := StartupCheck(cfg); err == nil {
		t.Errorf("expected StartupCheck to fail when targetDir does not exist")
	}

	// Create targetDir but missing subdirectories -> should fail
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := StartupCheck(cfg); err == nil {
		t.Errorf("expected StartupCheck to fail when subdirectories are missing")
	}

	// Create all required subdirectories (including anim_movie and anim_tv_series)
	for _, sub := range model.AllTargetDirs {
		if err := os.MkdirAll(filepath.Join(targetDir, string(sub)), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Now should succeed
	if err := StartupCheck(cfg); err != nil {
		t.Errorf("expected StartupCheck to succeed, got error: %v", err)
	}
	if cfg.Provider != "grok" {
		t.Errorf("expected provider grok, got %s", cfg.Provider)
	}
}
