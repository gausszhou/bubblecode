package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type ModelsList struct {
	Models      []string
	SelectedIdx int
	TitleStyle  lipgloss.Style
	ActiveStyle lipgloss.Style
	NormalStyle lipgloss.Style
}

func NewModelsList() ModelsList {
	return ModelsList{
		TitleStyle:  theme.SessionListTitleStyle,
		ActiveStyle: theme.SessionSelectStyle,
		NormalStyle: theme.SessionNormalStyle,
	}
}

func (ml ModelsList) View() string {
	var sb strings.Builder
	sb.WriteString(ml.TitleStyle.Render("Models"))
	sb.WriteString("\n")

	if len(ml.Models) == 0 {
		sb.WriteString(theme.SessionEmptyStyle.Render("No models"))
		return sb.String()
	}

	for i, m := range ml.Models {
		if i == ml.SelectedIdx {
			label := fmt.Sprintf("▸ %s", m)
			sb.WriteString(ml.ActiveStyle.Render("  " + label))
		} else {
			sb.WriteString(ml.NormalStyle.Render("    " + m))
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (ml *ModelsList) Up() {
	if ml.SelectedIdx > 0 {
		ml.SelectedIdx--
	}
}

func (ml *ModelsList) Down() {
	if ml.SelectedIdx < len(ml.Models)-1 {
		ml.SelectedIdx++
	}
}

func (ml ModelsList) OverlayView() string {
	bg := theme.ThemeBgOverlay
	ml.TitleStyle = ml.TitleStyle.Background(bg)
	ml.ActiveStyle = ml.ActiveStyle.Background(bg)
	ml.NormalStyle = ml.NormalStyle.Background(bg)

	content := ml.View()
	content += "\n\n"
	content += theme.HelpLabel().Background(bg).Render("↑/↓ navigate  •  Enter select  •  Esc close")
	return theme.OverlayBox().Render(content)
}
