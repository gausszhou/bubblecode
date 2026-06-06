package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Model struct {
	ID        string `json:"id"`
	MaxTokens *int   `json:"max_tokens,omitempty"`
}

type Provider struct {
	Name    string  `json:"name"`
	APIBase string  `json:"api_base"`
	APIKey  string  `json:"api_key"`
	Models  []Model `json:"models"`
}

func (p *Provider) MaxTokensVal(modelID string, globalMax int) int {
	for _, m := range p.Models {
		if m.ID == modelID && m.MaxTokens != nil {
			return *m.MaxTokens
		}
	}
	return globalMax
}

type Config struct {
	DefaultProvider string     `json:"default_provider"`
	DefaultModel    string     `json:"default_model"`
	Providers       []Provider `json:"providers"`
	MaxTokens       int        `json:"max_tokens"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers: []Provider{
			{
				Name:    "deepseek",
				APIBase: "https://api.deepseek.com/v1",
				Models:  []Model{{ID: "deepseek-chat"}, {ID: "deepseek-reasoner"}},
			},
		},
		DefaultProvider: "deepseek",
		DefaultModel:    "deepseek-chat",
		MaxTokens:       4096,
	}
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "bubblecode", "config.json"), nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	_, hasProviders := rawMap["providers"]
	_, hasAPIBase := rawMap["api_base"]

	if !hasProviders && (hasAPIBase || rawMap["api_key"] != nil) {
		var old struct {
			APIBase   string `json:"api_base"`
			APIKey    string `json:"api_key"`
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
		}
		if err := json.Unmarshal(data, &old); err != nil {
			return nil, fmt.Errorf("parse legacy config: %w", err)
		}

		model := old.Model
		if model == "" {
			model = "deepseek-chat"
		}
		apiKey := old.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("BUBBLECODE_API_KEY")
		}
		apiBase := old.APIBase
		if apiBase == "" {
			apiBase = "https://api.deepseek.com/v1"
		}
		maxTokens := old.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 4096
		}

		cfg := &Config{
			Providers: []Provider{
				{
					Name:    "default",
					APIBase: apiBase,
					APIKey:  apiKey,
					Models:  []Model{{ID: model}},
				},
			},
			DefaultProvider: "default",
			DefaultModel:    model,
			MaxTokens:       maxTokens,
		}

		_ = SaveConfig(path, cfg)

		return cfg, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.DefaultProvider == "" && len(cfg.Providers) > 0 {
		cfg.DefaultProvider = cfg.Providers[0].Name
	}
	if cfg.DefaultModel == "" {
		if p := cfg.GetDefaultProvider(); p != nil && len(p.Models) > 0 {
			cfg.DefaultModel = p.Models[0].ID
		} else {
			cfg.DefaultModel = "deepseek-chat"
		}
	}

	for i := range cfg.Providers {
		if cfg.Providers[i].APIKey == "" {
			cfg.Providers[i].APIKey = os.Getenv("BUBBLECODE_API_KEY")
		}
	}

	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) GetDefaultProvider() *Provider {
	for _, p := range c.Providers {
		if p.Name == c.DefaultProvider {
			return &p
		}
	}
	if len(c.Providers) > 0 {
		return &c.Providers[0]
	}
	return nil
}

func FetchModels(apiBase, apiKey string) ([]Model, error) {
	base := strings.TrimRight(apiBase, "/") + "/"
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(base),
		option.WithMiddleware(func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			req.Header.Set("User-Agent", "curl/7.81.0")
			return next(req)
		}),
	)

	page, err := client.Models.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list models: %v", err)
	}

	var models []Model
	for _, m := range page.Data {
		models = append(models, Model{ID: m.ID})
	}
	return models, nil
}

func (c *Config) SwitchProvider(name string) bool {
	for _, p := range c.Providers {
		if p.Name == name {
			c.DefaultProvider = name
			if len(p.Models) > 0 {
				c.DefaultModel = p.Models[0].ID
			}
			return true
		}
	}
	return false
}
