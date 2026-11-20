package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAIModel = "nvidia/nemotron-3.5-lightning-30b-a3b"
	appDirName     = "go-vit"
	configFileName = "config.json"
)

// Config stores user-level persistent settings for go-vit.
// Sensitive values like AIAPIKey are stored with 0600 permissions.
type Config struct {
	AIAPIKey string `json:"ai_api_key,omitempty"`
	AIModel  string `json:"ai_model,omitempty"`
}

// ConfigPath returns the path to the user's go-vit config file.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName, configFileName), nil
}

// Load reads the config file from disk (if present) and applies environment overrides.
func Load() (Config, error) {
	var cfg Config
	path, err := ConfigPath()
	if err != nil {
		return applyEnv(cfg), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return applyEnv(cfg), nil
		}
		return cfg, err
	}

	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}

	return applyEnv(cfg), nil
}

func applyEnv(cfg Config) Config {
	if key := strings.TrimSpace(os.Getenv("AI_API_KEY")); key != "" {
		cfg.AIAPIKey = key
	}
	if model := strings.TrimSpace(os.Getenv("GO_VIT_AI_MODEL")); model != "" {
		cfg.AIModel = model
	}
	if strings.TrimSpace(cfg.AIModel) == "" {
		cfg.AIModel = DefaultAIModel
	}
	return cfg
}

// Save persists the config to disk with restrictive permissions (0700 dir, 0600 file).
func Save(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// MaskKey returns a safely masked representation of an API key for display.
func MaskKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "(não configurada)"
	}
	if len(key) <= 10 {
		return "****"
	}
	return key[:6] + "..." + key[len(key)-4:]
}
