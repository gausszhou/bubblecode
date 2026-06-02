package bubblecode

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
)

var configPath string

func acpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Start as an ACP server (stdio mode)",
		Long:  `Start the LLM agent as an ACP server communicating over stdin/stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runACPServer(cmd)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file (default ~/.config/bubblecode/config.json)")

	return cmd
}

func runACPServer(cmd *cobra.Command) error {
	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelInfo}))

	path := configPath
	if path == "" {
		var err error
		path, err = agent.ConfigPath()
		if err != nil {
			return fmt.Errorf("config path: %w", err)
		}
	}

	cfg, err := agent.LoadConfig(path)
	if err != nil {
		logger.Warn("config not loaded, using defaults", "error", err)
		cfg = agent.DefaultConfig()
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured: set BUBBLECODE_API_KEY env var or create config at %s", path)
	}

	agentInstance := agent.NewLLMAgent(cfg, logger)
	conn := acp.NewAgentSideConnection(agentInstance, os.Stdout, os.Stdin)
	agentInstance.SetAgentConnection(conn)

	logger.Info("ACP server started", "model", cfg.Model)

	<-conn.Done()
	logger.Info("ACP server stopped")

	return nil
}
