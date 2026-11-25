package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://integrate.api.nvidia.com/v1/chat/completions"
	defaultMaxDiff = 8000
	httpTimeout    = 30 * time.Second
)

var (
	// ErrUnauthorized is returned when the AI API rejects the configured API key.
	ErrUnauthorized = errors.New("chave de API inválida ou sem permissão (HTTP 401/403)")

	noisyLockfiles = map[string]bool{
		"go.sum":            true,
		"package-lock.json": true,
		"yarn.lock":         true,
		"pnpm-lock.yaml":    true,
		"Cargo.lock":        true,
	}
)

// Client calls NVIDIA NIM's OpenAI-compatible chat completions API.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new NVIDIA NIM AI client.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

type chatRequest struct {
	Model              string         `json:"model"`
	Messages           []chatMessage  `json:"messages"`
	Temperature        float64        `json:"temperature"`
	MaxTokens          int            `json:"max_tokens"`
	ChatTemplateKwargs map[string]any `json:"chat_template_kwargs,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateCommitMessage generates a Conventional Commit message from a staged git diff.
func (c *Client) GenerateCommitMessage(ctx context.Context, stagedDiff string) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("chave de API de IA não configurada")
	}

	cleaned := FilterAndTruncateDiff(stagedDiff, defaultMaxDiff)
	if strings.TrimSpace(cleaned) == "" {
		return "", errors.New("nenhuma alteração preparada (staged) para analisar")
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You are an expert Git assistant. Generate ONLY a single-line Conventional Commit message in English without a scope (e.g. feat: ..., fix: ..., refactor: ...). Do NOT include a scope in parentheses. Be specific about what changed and why based on the diff (include key symbol, file, or behavior names). Keep it under 90 characters. Do not use backticks, quotes, explanations, or markdown.",
			},
			{
				Role:    "user",
				Content: "Generate the commit message for this git diff:\n\n" + cleaned,
			},
		},
		Temperature: 0.2,
		MaxTokens:   200,
		ChatTemplateKwargs: map[string]any{
			"enable_thinking": false,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar requisição: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha ao conectar na API de IA: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("erro ao ler resposta da API: %w", err)
	}

	var parsed chatResponse
	_ = json.Unmarshal(body, &parsed)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("erro da API (%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return "", fmt.Errorf("erro da API (HTTP %d)", resp.StatusCode)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("a API não retornou nenhuma sugestão de commit")
	}

	msg := sanitizeCommitMessage(parsed.Choices[0].Message.Content)
	if msg == "" {
		return "", fmt.Errorf("a API retornou uma mensagem vazia (finish_reason=%s)", parsed.Choices[0].FinishReason)
	}
	return msg, nil
}

// FilterAndTruncateDiff removes noisy lockfile sections and truncates the diff to maxChars.
func FilterAndTruncateDiff(diff string, maxChars int) string {
	if strings.TrimSpace(diff) == "" {
		return ""
	}

	sections := strings.Split(diff, "diff --git ")
	var kept []string
	for _, sec := range sections {
		if strings.TrimSpace(sec) == "" {
			continue
		}
		firstLine, _, _ := strings.Cut(sec, "\n")
		fields := strings.Fields(firstLine)
		skip := false
		if len(fields) >= 2 {
			bPath := strings.TrimPrefix(fields[1], "b/")
			if noisyLockfiles[filepath.Base(bPath)] {
				skip = true
			}
		}
		if !skip {
			kept = append(kept, "diff --git "+sec)
		}
	}

	out := strings.Join(kept, "")
	if maxChars > 0 && len(out) > maxChars {
		out = out[:maxChars] + "\n\n[... diff truncado ...]"
	}
	return out
}

func sanitizeCommitMessage(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`+"`")
	// Keep only the first non-empty line for single-line commit input.
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
