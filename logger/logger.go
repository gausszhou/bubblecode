package logger

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	ComponentAgent  = "agent"
	ComponentClient = "client"
)

type Config struct {
	Level     slog.Level
	AddSource bool
	Dir       string
}

func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".gausszhou", "bubblecode", "logs")
}

func DefaultConfig() Config {
	return Config{Level: slog.LevelInfo}
}

func New(component string, cfg Config) (*slog.Logger, error) {
	dir := cfg.Dir
	if dir == "" {
		dir = defaultDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, component+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	})), nil
}
