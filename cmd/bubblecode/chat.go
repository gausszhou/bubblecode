package bubblecode

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
	"github.com/gausszhou/bubblecode/client"
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
	if !contains(existing, base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !contains(existing, candidate) {
			return candidate
		}
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func promptProvider(reader *bufio.Reader, num int, existingNames ...string) (agent.Provider, error) {
	var p agent.Provider

	fmt.Println("  Select preset provider:")
	for i, pr := range presets {
		fmt.Printf("    %d) %s\n", i+1, pr.name)
	}
	fmt.Printf("    %d) custom\n", len(presets)+1)
	fmt.Printf("  Choice [%d]: ", len(presets)+1)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	idx, err := strconv.Atoi(choice)
	if err == nil && idx >= 1 && idx <= len(presets) {
		pr := presets[idx-1]
		p.Name = pr.name
		p.APIBase = pr.base
	} else {
		fmt.Printf("  Name [provider-%d]: ", num)
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = fmt.Sprintf("provider-%d", num)
		}
		p.Name = name

		defaultBase := "https://api.deepseek.com/v1"
		fmt.Printf("  API Base URL [%s]: ", defaultBase)
		base, _ := reader.ReadString('\n')
		base = strings.TrimSpace(base)
		if base == "" {
			base = defaultBase
		}
		p.APIBase = base
	}

	p.Name = uniqueName(p.Name, existingNames)
	fmt.Printf("    Name:     %s\n", p.Name)
	fmt.Printf("    API Base: %s\n", p.APIBase)

	for {
		fmt.Printf("  API Key: ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key != "" {
			p.APIKey = key
			break
		}
		fmt.Println("    API Key cannot be empty.")
	}

	fmt.Println("  Fetching available models...")
	apiModels, err := agent.FetchModels(p.APIBase, p.APIKey)
	if err != nil {
		fmt.Printf("  Warning: could not fetch models (%v)\n", err)
		fmt.Printf("  Models (comma-separated) [deepseek-chat]: ")
		modelsStr, _ := reader.ReadString('\n')
		modelsStr = strings.TrimSpace(modelsStr)
		if modelsStr == "" {
			p.Models = []string{"deepseek-chat"}
		} else {
			for _, m := range strings.Split(modelsStr, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					p.Models = append(p.Models, m)
				}
			}
		}
	} else {
		for i, m := range apiModels {
			fmt.Printf("    %d. %s\n", i+1, m)
		}
		fmt.Printf("  Select models by number (comma-separated, Enter for all): ")
		sel, _ := reader.ReadString('\n')
		sel = strings.TrimSpace(sel)
		if sel == "" {
			p.Models = apiModels
		} else {
			for _, s := range strings.Split(sel, ",") {
				s = strings.TrimSpace(s)
				idx, err := strconv.Atoi(s)
				if err == nil && idx >= 1 && idx <= len(apiModels) {
					p.Models = append(p.Models, apiModels[idx-1])
				}
			}
			if len(p.Models) == 0 {
				p.Models = apiModels
			}
		}
	}

	return p, nil
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

	reader := bufio.NewReader(os.Stdin)

	var providers []agent.Provider
	for i := 0; ; i++ {
		fmt.Printf("Provider %d:\n", i+1)
		var existingNames []string
		for _, pr := range providers {
			existingNames = append(existingNames, pr.Name)
		}
		p, err := promptProvider(reader, i+1, existingNames...)
		if err != nil {
			return nil, err
		}
		providers = append(providers, p)

		fmt.Print("Add another provider? [y/N]: ")
		more, _ := reader.ReadString('\n')
		more = strings.TrimSpace(more)
		if strings.ToLower(more) != "y" {
			break
		}
		fmt.Println()
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

	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	conn, err := client.NewConnection(agentCmd, acpClient, logger)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	initResp, err := conn.Initialize(ctx, acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ClientCapabilities: acpsdk.ClientCapabilities{
			Fs:       acpsdk.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
			Terminal: true,
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	logger.Info("agent initialized", "protocol_version", initResp.ProtocolVersion)

	newSess, err := conn.NewSession(ctx, acpsdk.NewSessionRequest{
		Cwd:        client.MustCwd(),
		McpServers: []acpsdk.McpServer{},
	})
	if err != nil {
		return fmt.Errorf("new session failed: %w", err)
	}
	logger.Info("session created", "session_id", newSess.SessionId)

	inputCh := make(chan client.InputCommand, 1)

	cl := client.NewClient(inputCh, conn.ClientConn(), events)
	go cl.Run(ctx, newSess.SessionId)

	model := tui.NewModel(logger, agentCmd, string(newSess.SessionId), ctx, cancel, inputCh, events)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
