package bubblecode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	tea "charm.land/bubbletea/v2"
	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
	"github.com/gausszhou/bubblecode/client"
	"github.com/gausszhou/bubblecode/logger"
	"github.com/gausszhou/bubblecode/tui"
)

type preset struct {
	name string
	base string
}

var presets = []preset{
	{name: "gausszhou", base: "https://mock.gausszhou.top/openai/v1"},
	{name: "deepseek", base: "https://api.deepseek.com/v1"},
}

func uniqueName(base string, existing []string) string {
	if !slices.Contains(existing, base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !slices.Contains(existing, candidate) {
			return candidate
		}
	}
}

func promptProvider(num int, existingNames ...string) (agent.Provider, error) {
	var name, apiBase string
	var choice string

	presetOpts := make([]huh.Option[string], 0, len(presets)+1)
	for _, pr := range presets {
		presetOpts = append(presetOpts, huh.NewOption(pr.name, pr.name))
	}
	presetOpts = append(presetOpts, huh.NewOption("custom", "__custom__"))

	err := huh.NewSelect[string]().
		Title("Select preset provider").
		Options(presetOpts...).
		Value(&choice).
		Run()
	if err != nil {
		return agent.Provider{}, err
	}

	if choice != "__custom__" {
		for _, pr := range presets {
			if pr.name == choice {
				name = pr.name
				apiBase = pr.base
				break
			}
		}
	} else {
		defaultName := fmt.Sprintf("provider-%d", num)
		err = huh.NewInput().
			Title("Provider name").
			Value(&name).
			Placeholder(defaultName).
			Run()
		if err != nil {
			return agent.Provider{}, err
		}
		if name == "" {
			name = defaultName
		}

		defaultBase := "https://api.deepseek.com/v1"
		err = huh.NewInput().
			Title("API Base URL").
			Value(&apiBase).
			Placeholder(defaultBase).
			Run()
		if err != nil {
			return agent.Provider{}, err
		}
		if apiBase == "" {
			apiBase = defaultBase
		}
	}

	name = uniqueName(name, existingNames)

	var apiKey string
	err = huh.NewInput().
		Title(fmt.Sprintf("API Key for %s", name)).
		EchoMode(huh.EchoModePassword).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("API Key cannot be empty")
			}
			return nil
		}).
		Value(&apiKey).
		Run()
	if err != nil {
		return agent.Provider{}, err
	}

	fmt.Println("  Fetching available models...")
	apiModels, fetchErr := agent.FetchModels(apiBase, apiKey)

	var models []string
	if fetchErr != nil {
		fmt.Printf("  Warning: could not fetch models (%v)\n", fetchErr)
		var modelsInput string
		err = huh.NewInput().
			Title("Models (comma-separated)").
			Placeholder("deepseek-chat").
			Value(&modelsInput).
			Run()
		if err != nil {
			return agent.Provider{}, err
		}
		if modelsInput == "" {
			models = []string{"deepseek-chat"}
		} else {
			for _, m := range splitComma(modelsInput) {
				models = append(models, m)
			}
		}
	} else {
		modelOpts := make([]huh.Option[string], len(apiModels))
		for i, m := range apiModels {
			modelOpts[i] = huh.NewOption(m, m)
		}
		var selected []string
		err = huh.NewMultiSelect[string]().
			Title("Select models (space to toggle, enter to confirm)").
			Options(modelOpts...).
			Value(&selected).
			Run()
		if err != nil {
			return agent.Provider{}, err
		}
		if len(selected) == 0 {
			models = apiModels
		} else {
			models = selected
		}
	}

	return agent.Provider{
		Name:    name,
		APIBase: apiBase,
		APIKey:  apiKey,
		Models:  models,
	}, nil
}

func splitComma(s string) []string {
	var result []string
	for _, m := range strings.Split(s, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			result = append(result, m)
		}
	}
	return result
}

func ensureConfig() (*agent.Config, error) {
	path, err := agent.ConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := agent.LoadConfig(path)
	if err == nil {
		p := cfg.GetActiveProvider()
		if p != nil && p.APIKey != "" {
			return cfg, nil
		}
	}

	if envKey := os.Getenv("BUBBLECODE_API_KEY"); envKey != "" {
		cfg = agent.DefaultConfig()
		cfg.Providers[0].APIKey = envKey
		if saveErr := agent.SaveConfig(path, cfg); saveErr == nil {
			fmt.Println("Config saved to", path)
		}
		return cfg, nil
	}

	fmt.Println("━━━ Setup ━━━")
	fmt.Println("No API configuration found. Please add at least one provider.")
	fmt.Println()

	var providers []agent.Provider
	for i := 0; ; i++ {
		var existingNames []string
		for _, pr := range providers {
			existingNames = append(existingNames, pr.Name)
		}
		p, err := promptProvider(i+1, existingNames...)
		if err != nil {
			return nil, err
		}
		providers = append(providers, p)

		var more bool
		err = huh.NewConfirm().
			Title("Add another provider?").
			Value(&more).
			Run()
		if err != nil {
			return nil, err
		}
		if !more {
			break
		}
	}

	cfg = &agent.Config{
		Providers:      providers,
		ActiveProvider: providers[0].Name,
		ActiveModel:    providers[0].Models[0],
		MaxTokens:      4096,
		Temperature:    0.7,
	}

	if err := agent.SaveConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	fmt.Println("Config saved to", path)
	fmt.Println()

	return cfg, nil
}

func runChat(cmd *cobra.Command) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, err := logger.New(logger.ComponentClient, logger.DefaultConfig())
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	if _, err := ensureConfig(); err != nil {
		return fmt.Errorf("config setup failed: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	agentCmd := exec.CommandContext(ctx, execPath, "acp")

	events := make(chan client.OutputEvent, 100)

	acpClient := client.NewACPClient(events)

	conn, err := client.NewConnection(agentCmd, acpClient, log)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	initResp, err := conn.Initialize(ctx, acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ClientCapabilities: acpsdk.ClientCapabilities{
			Fs: acpsdk.FileSystemCapabilities{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	log.Info("agent initialized", "protocol_version", initResp.ProtocolVersion)

	newSess, err := conn.NewSession(ctx, acpsdk.NewSessionRequest{
		Cwd:        client.MustCwd(),
		McpServers: []acpsdk.McpServer{},
	})
	if err != nil {
		return fmt.Errorf("new session failed: %w", err)
	}
	log.Info("session created", "session_id", newSess.SessionId)

	inputCh := make(chan client.InputCommand, 1)

	cl := client.NewClient(inputCh, conn.ClientConn(), events)
	go cl.Run(ctx, newSess.SessionId)

	model := tui.NewModel(log, agentCmd, string(newSess.SessionId), ctx, cancel, inputCh, events)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
