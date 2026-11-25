package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FelipeSrutkoske/go-vit/internal/config"
)

func TestRunConfig_SetsAndUpdatesAPIKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AI_API_KEY", "")

	// 1. No key yet -> prompts and saves
	var out bytes.Buffer
	if err := runConfig(strings.NewReader("nvapi-first-key-9999\n"), &out); err != nil {
		t.Fatalf("runConfig failed: %v", err)
	}
	if !strings.Contains(out.String(), "Nenhuma chave de API") || !strings.Contains(out.String(), "Chave de API salva") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	cfg, err := config.Load()
	if err != nil || cfg.AIAPIKey != "nvapi-first-key-9999" {
		t.Fatalf("expected saved key, got %q (err=%v)", cfg.AIAPIKey, err)
	}

	// 2. Existing key -> decline replacement
	out.Reset()
	if err := runConfig(strings.NewReader("n\n"), &out); err != nil {
		t.Fatalf("runConfig decline failed: %v", err)
	}
	if !strings.Contains(out.String(), "Nenhuma alteração realizada") {
		t.Fatalf("expected no-change message, got: %s", out.String())
	}

	// 3. Existing key -> accept replacement
	out.Reset()
	if err := runConfig(strings.NewReader("s\nnvapi-second-key-1111\n"), &out); err != nil {
		t.Fatalf("runConfig replace failed: %v", err)
	}
	cfg, err = config.Load()
	if err != nil || cfg.AIAPIKey != "nvapi-second-key-1111" {
		t.Fatalf("expected updated key, got %q (err=%v)", cfg.AIAPIKey, err)
	}
}
