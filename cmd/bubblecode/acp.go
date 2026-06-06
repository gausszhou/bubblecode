package bubblecode

import (
	"fmt"
	"os"

	"github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
	"github.com/gausszhou/bubblecode/logger"
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
	log, err := logger.New(logger.ComponentAgent, logger.DefaultConfig())
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

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
		log.Warn("config not loaded, using defaults", "error", err)
		cfg = agent.DefaultConfig()
	}

	p := cfg.GetActiveProvider()
	if p == nil || p.APIKey == "" {
		log.Warn("API key not configured, agent will return errors until configured",
			"hint", "set BUBBLECODE_API_KEY env var or create config at "+path)
	}

	agentInstance := agent.NewLLMAgent(cfg, log)
	conn := acp.NewAgentSideConnection(agentInstance, os.Stdout, os.Stdin)
	agentInstance.SetAgentConnection(conn)

	log.Info("ACP server started", "provider", cfg.ActiveProvider, "model", cfg.ActiveModel)

	<-conn.Done()
	log.Info("ACP server stopped")

	return nil
}
