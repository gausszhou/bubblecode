package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type SessionItem struct {
	ID     string
	Name   string
	Active bool
}

type SessionList struct {
	Sessions    []SessionItem
	Title       string
	SelectedIdx int
	TitleStyle  lipgloss.Style
	ActiveStyle lipgloss.Style
	NormalStyle lipgloss.Style
	SelectStyle lipgloss.Style
}

func NewSessionList(title string) SessionList {
	return SessionList{
		Title:       title,
		TitleStyle:  theme.SessionListTitleStyle,
		ActiveStyle: theme.SessionActiveStyle,
		NormalStyle: theme.SessionNormalStyle,
		SelectStyle: theme.SessionSelectStyle,
	}
}

func (sl SessionList) View() string {
	var sb strings.Builder
	sb.WriteString(sl.TitleStyle.Render(sl.Title))
	sb.WriteString("\n")

	if len(sl.Sessions) == 0 {
		sb.WriteString(theme.SessionEmptyStyle.Render("No sessions"))
		return sb.String()
	}

	for i, sess := range sl.Sessions {
		marker := ""
		if sess.Active {
			marker = "●"
		}

		if i == sl.SelectedIdx {
			label := fmt.Sprintf("▶ %s %s", marker, truncate(sess.Name, 28))
			sb.WriteString(sl.SelectStyle.Render(label))
		} else {
			prefix := "   "
			label := fmt.Sprintf("%s %s", marker, truncate(sess.Name, 28))
			if sess.Active {
				sb.WriteString(sl.ActiveStyle.Render(prefix + label))
			} else {
				sb.WriteString(sl.NormalStyle.Render(prefix + label))
			}
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (sl *SessionList) Up() {
	if sl.SelectedIdx > 0 {
		sl.SelectedIdx--
	}
}

func (sl *SessionList) Down() {
	if sl.SelectedIdx < len(sl.Sessions)-1 {
		sl.SelectedIdx++
	}
}

func (sl SessionList) OverlayView() string {
	bg := theme.ThemeBgOverlay
	sl.TitleStyle = sl.TitleStyle.Background(bg)
	sl.ActiveStyle = sl.ActiveStyle.Background(bg)
	sl.NormalStyle = sl.NormalStyle.Background(bg)
	sl.SelectStyle = sl.SelectStyle.Background(bg)

	content := sl.View()
	content += "\n\n"
	content += theme.HelpLabel().Background(bg).Render("↑/↓ navigate  •  Enter switch  •  Esc close")
	return theme.OverlayBox().Render(content)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
