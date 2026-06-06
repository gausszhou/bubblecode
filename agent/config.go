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

type Provider struct {
	Name    string   `json:"name"`
	APIBase string   `json:"api_base"`
	APIKey  string   `json:"api_key"`
	Models  []string `json:"models"`
}

type Config struct {
	ActiveProvider string     `json:"active_provider"`
	ActiveModel    string     `json:"active_model"`
	Providers      []Provider `json:"providers"`
	MaxTokens      int        `json:"max_tokens"`
	Temperature    float64    `json:"temperature"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers: []Provider{
			{
				Name:    "deepseek",
				APIBase: "https://api.deepseek.com/v1",
				Models:  []string{"deepseek-chat", "deepseek-reasoner"},
			},
		},
		ActiveProvider: "deepseek",
		ActiveModel:    "deepseek-chat",
		MaxTokens:      4096,
		Temperature:    0.7,
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
			APIBase     string  `json:"api_base"`
			APIKey      string  `json:"api_key"`
			Model       string  `json:"model"`
			MaxTokens   int     `json:"max_tokens"`
			Temperature float64 `json:"temperature"`
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
		temp := old.Temperature
		if temp <= 0 {
			temp = 0.7
		}

		cfg := &Config{
			Providers: []Provider{
				{
					Name:    "default",
					APIBase: apiBase,
					APIKey:  apiKey,
					Models:  []string{model},
				},
			},
			ActiveProvider: "default",
			ActiveModel:    model,
			MaxTokens:      maxTokens,
			Temperature:    temp,
		}

		if saveErr := SaveConfig(path, cfg); saveErr == nil {
			// migrated
		}

		return cfg, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Temperature <= 0 {
		cfg.Temperature = 0.7
	}
	if cfg.ActiveProvider == "" && len(cfg.Providers) > 0 {
		cfg.ActiveProvider = cfg.Providers[0].Name
	}
	if cfg.ActiveModel == "" {
		if p := cfg.GetActiveProvider(); p != nil && len(p.Models) > 0 {
			cfg.ActiveModel = p.Models[0]
		} else {
			cfg.ActiveModel = "deepseek-chat"
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
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) GetActiveProvider() *Provider {
	for _, p := range c.Providers {
		if p.Name == c.ActiveProvider {
			return &p
		}
	}
	if len(c.Providers) > 0 {
		return &c.Providers[0]
	}
	return nil
}

func FetchModels(apiBase, apiKey string) ([]string, error) {
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

	var models []string
	for _, m := range page.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func (c *Config) SwitchProvider(name string) bool {
	for _, p := range c.Providers {
		if p.Name == name {
			c.ActiveProvider = name
			if len(p.Models) > 0 {
				c.ActiveModel = p.Models[0]
			}
			return true
		}
	}
	return false
}
