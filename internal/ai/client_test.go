package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFilterAndTruncateDiff(t *testing.T) {
	diff := "diff --git a/go.sum b/go.sum\n+lots of hashes\ndiff --git a/main.go b/main.go\n+fmt.Println(\"hi\")\n"
	got := FilterAndTruncateDiff(diff, 500)
	if strings.Contains(got, "go.sum") {
		t.Fatalf("expected go.sum to be filtered out, got: %s", got)
	}
	if !strings.Contains(got, "main.go") {
		t.Fatalf("expected main.go to be kept, got: %s", got)
	}

	long := strings.Repeat("a", 100)
	trunc := FilterAndTruncateDiff("diff --git a/a.go b/a.go\n"+long, 30)
	if !strings.Contains(trunc, "[... diff truncado ...]") {
		t.Fatalf("expected truncation marker, got: %s", trunc)
	}
}

func TestGenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer nvapi-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "`feat: adiciona gerador de commit com IA`"}},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("nvapi-test", "meta/llama-3.1-8b-instruct")
	c.baseURL = srv.URL

	msg, err := c.GenerateCommitMessage(context.Background(), "diff --git a/main.go b/main.go\n+hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: adiciona gerador de commit com IA" {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestGenerateCommitMessage_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("nvapi-bad", "meta/llama-3.1-8b-instruct")
	c.baseURL = srv.URL

	_, err := c.GenerateCommitMessage(context.Background(), "diff --git a/a.go b/a.go\n+x")
	if err == nil || !strings.Contains(err.Error(), "401/403") {
		t.Fatalf("expected 401/403 error, got: %v", err)
	}
}
