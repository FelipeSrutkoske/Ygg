package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad_PermissionsAndEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("AI_API_KEY", "")
	t.Setenv("GO_VIT_AI_MODEL", "")

	want := Config{
		AIAPIKey: "api-secret-1234567890",
		AIModel:  "meta/llama-3.1-8b-instruct",
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cfgPath := filepath.Join(tmpDir, appDirName, configFileName)
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected file perm 0600, got %o", perm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got.AIAPIKey != want.AIAPIKey {
		t.Fatalf("expected key %q, got %q", want.AIAPIKey, got.AIAPIKey)
	}

	t.Setenv("AI_API_KEY", "api-env-override")
	gotEnv, err := Load()
	if err != nil {
		t.Fatalf("Load with env failed: %v", err)
	}
	if gotEnv.AIAPIKey != "api-env-override" {
		t.Fatalf("expected env override, got %q", gotEnv.AIAPIKey)
	}
}

func TestMaskKey(t *testing.T) {
	if got := MaskKey(""); got != "(não configurada)" {
		t.Fatalf("unexpected mask for empty: %q", got)
	}
	if got := MaskKey("short"); got != "****" {
		t.Fatalf("unexpected mask for short: %q", got)
	}
	if got := MaskKey("nvapi-1234567890abcd"); got != "nvapi-...abcd" {
		t.Fatalf("unexpected mask: %q", got)
	}
}
