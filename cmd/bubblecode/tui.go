package bubblecode

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/client"
	"github.com/gausszhou/bubblecode/logger"
	"github.com/gausszhou/bubblecode/tui"
)

func runTUI(cmd *cobra.Command) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, err := logger.New(logger.ComponentClient, logger.DefaultConfig())
	if err != nil {
		return fmt.Errorf("logger: %w", err)
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
			Fs:       acpsdk.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
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
