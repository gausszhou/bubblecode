package component

import (
	"fmt"
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

func (cp CommandPanel) OverlayView(selected int) string {
	bg := theme.ThemeBgOverlay
	var sb strings.Builder
	sb.WriteString(theme.AccentStyle().Background(bg).Render("Commands"))
	sb.WriteString("\n\n")
	for i, cmd := range cp.Commands {
		mark := "  "
		keyStyle := theme.CommandKeyStyle
		descStyle := theme.CommandDescStyle
		if i == selected {
			mark = "▸ "
			keyStyle = theme.AccentStyle()
			descStyle = theme.TextStyle()
		}
		sb.WriteString("  ")
		sb.WriteString(mark)
		sb.WriteString(keyStyle.Background(bg).Render(cmd.Key))
		sb.WriteString("  ")
		sb.WriteString(descStyle.Background(bg).Render(cmd.Desc))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(theme.AccentStyle().Background(bg).Render("Slash commands in chat"))
	sb.WriteString("\n\n")
	for _, sc := range SlashCommands {
		sb.WriteString("  ")
		sb.WriteString(theme.CommandKeyStyle.Background(bg).Render(sc.Command))
		sb.WriteString("  ")
		sb.WriteString(theme.CommandDescStyle.Background(bg).Render(sc.Desc))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	help := fmt.Sprintf("%s  %s  %s",
		theme.CommandKeyStyle.Background(bg).Render("↑/↓"),
		theme.CommandDescStyle.Background(bg).Render("navigate"),
		theme.HelpLabel().Background(bg).Render("•"),
	)
	help += fmt.Sprintf("  %s  %s",
		theme.CommandKeyStyle.Background(bg).Render("Enter"),
		theme.CommandDescStyle.Background(bg).Render("select"),
	)
	help += fmt.Sprintf("  %s  %s",
		theme.CommandKeyStyle.Background(bg).Render("Esc"),
		theme.CommandDescStyle.Background(bg).Render("close"),
	)
	sb.WriteString(help)
	return theme.OverlayBox().Render(sb.String())
}
