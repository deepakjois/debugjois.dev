package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLocalEnvFileRequiresDeepgramAndLinkPreview(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	err := os.WriteFile(envPath, []byte("LINKPREVIEW_API_KEY=linkpreview-key\nDEEPGRAM_API_KEY=deepgram-key\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	t.Setenv(linkPreviewAPIKeyEnvVar, "")
	t.Setenv(deepgramAPIKeyEnvVar, "")

	if err := loadLocalEnvFile(); err != nil {
		t.Fatalf("loadLocalEnvFile returned error: %v", err)
	}
	if got := os.Getenv(linkPreviewAPIKeyEnvVar); got != "linkpreview-key" {
		t.Fatalf("expected link preview env to be loaded, got %q", got)
	}
	if got := os.Getenv(deepgramAPIKeyEnvVar); got != "deepgram-key" {
		t.Fatalf("expected deepgram env to be loaded, got %q", got)
	}
}

func TestLoadLocalEnvFileRequiresDeepgramAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	err := os.WriteFile(envPath, []byte("LINKPREVIEW_API_KEY=linkpreview-key\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	t.Setenv(linkPreviewAPIKeyEnvVar, "")
	t.Setenv(deepgramAPIKeyEnvVar, "")

	err = loadLocalEnvFile()
	if err == nil {
		t.Fatal("expected loadLocalEnvFile to fail without DEEPGRAM_API_KEY")
	}
	if !strings.Contains(err.Error(), deepgramAPIKeyEnvVar) {
		t.Fatalf("expected error to mention %s, got %v", deepgramAPIKeyEnvVar, err)
	}
}
