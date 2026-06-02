package component

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type Command struct {
	Key  string
	Desc string
}

type CommandPanel struct {
	Commands  []Command
	Style     lipgloss.Style
	KeyStyle  lipgloss.Style
	DescStyle lipgloss.Style
}

func NewCommandPanel(commands []Command) CommandPanel {
	return CommandPanel{
		Commands:  commands,
		Style:     theme.CommandPanelStyle,
		KeyStyle:  theme.CommandKeyStyle,
		DescStyle: theme.CommandDescStyle,
	}
}

func (cp CommandPanel) View() string {
	var sb strings.Builder
	for _, cmd := range cp.Commands {
		sb.WriteString(cp.KeyStyle.Render(cmd.Key))
		sb.WriteString(" ")
		sb.WriteString(cp.DescStyle.Render(cmd.Desc))
		sb.WriteString("  ")
	}
	return cp.Style.Render(strings.TrimRight(sb.String(), " "))
}

func DefaultCommands() CommandPanel {
	return NewCommandPanel([]Command{
		{Key: "Enter", Desc: "Send"},
		{Key: "Shift+Enter", Desc: "Newline"},
		{Key: "Esc Esc", Desc: "Interrupt"},
		{Key: "Ctrl+P", Desc: "Commands"},
		{Key: "Ctrl+M", Desc: "Change Model"},
		{Key: "Ctrl+S", Desc: "Switch Session"},
		{Key: "Ctrl+N", Desc: "New Session"},
		{Key: "Ctrl+C", Desc: "Quit"},
	})
}
