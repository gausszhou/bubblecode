package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	flex "github.com/gausszhou/bubbleflex"

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
		{Key: "Ctrl+N", Desc: "New Session"},
		{Key: "Ctrl+S", Desc: "Switch Session"},
		{Key: "Ctrl+M", Desc: "Change Model"},
	})
}

func (cp CommandPanel) OverlayView(selected int) string {
	bg := theme.ThemeBgOverlay
	accent := theme.AccentStyle().Background(bg)
	keyStyle := theme.CommandKeyStyle.Background(bg)
	descStyle := theme.CommandDescStyle.Background(bg)
	helpLabel := theme.HelpLabel().Background(bg)
	indent := lipgloss.NewStyle().PaddingLeft(2).Background(bg)

	content := accent.Render("Commands") + "\n\n"
	for i, cmd := range cp.Commands {
		rowDesc := descStyle
		if i == selected {
			rowDesc = theme.BaseStyle().Background(theme.ThemeWarning)
		}
		desc := rowDesc.Render("  " + cmd.Desc)
		key := keyStyle.Render(cmd.Key)
		content += indent.Render(flex.New(flex.Row).
			JustifyContent(flex.SpaceBetween).
			Width(52).
			Gap(2).
			Join(desc, key))
		content += "\n"
	}
	content += "\n"
	help := fmt.Sprintf("%s  %s  %s",
		keyStyle.Render("↑/↓"),
		descStyle.Render("navigate"),
		helpLabel.Render("•"),
	)
	help += fmt.Sprintf("  %s  %s",
		keyStyle.Render("Enter"),
		descStyle.Render("select"),
	)
	help += fmt.Sprintf("  %s  %s",
		keyStyle.Render("Esc"),
		descStyle.Render("close"),
	)
	content += indent.Render(help)
	return theme.OverlayBox().Render(content)
}
