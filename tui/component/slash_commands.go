package component

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/gausszhou/bubblecode/tui/theme"
)

type SlashCommand struct {
	Command string
	Desc    string
}

var SlashCommands = []SlashCommand{
	{Command: "/new", Desc: "Create new session"},
	{Command: "/models", Desc: "Switch current model"},
	{Command: "/sessions", Desc: "Switch current session"},
}

func MatchingSlashCommands(input string) []SlashCommand {
	if !strings.HasPrefix(input, "/") || strings.TrimSpace(input) == "/" {
		return SlashCommands
	}
	lower := strings.ToLower(input)
	var matches []SlashCommand
	for _, sc := range SlashCommands {
		cmdName := strings.Split(sc.Command, " ")[0]
		if strings.HasPrefix(strings.ToLower(cmdName), lower) {
			matches = append(matches, sc)
		}
	}
	if len(matches) == 0 {
		for _, sc := range SlashCommands {
			cmdName := strings.Split(sc.Command, " ")[0]
			if strings.Contains(strings.ToLower(cmdName), lower) {
				matches = append(matches, sc)
			}
		}
	}
	return matches
}

func SelectCommandIndex(matches []SlashCommand, idx int) *SlashCommand {
	if idx < 0 || idx >= len(matches) {
		return nil
	}
	return &matches[idx]
}

func SelectCommandName(matches []SlashCommand, name string) *SlashCommand {
	for i := range matches {
		if matches[i].Command == name {
			return &matches[i]
		}
	}
	return nil
}

func RenderSlashSuggestions(input string, width int) string {
	matches := MatchingSlashCommands(input)
	if len(matches) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, sc := range matches {
		sb.WriteString(theme.CommandKeyStyle.Render(sc.Command))
		sb.WriteString("  ")
		sb.WriteString(theme.CommandDescStyle.Render(sc.Desc))
		sb.WriteString("\n")
	}
	content := strings.TrimRight(sb.String(), "\n")
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(theme.ThemeAccent).
		Background(theme.ThemeSurface).
		Padding(0, 1).
		Render(content)
}
