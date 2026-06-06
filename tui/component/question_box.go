package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type QuestionBox struct {
	Title         string
	Message       string
	Options       []string
	SelectedIdx   int
	Width         int
	Style         lipgloss.Style
	ActiveStyle   lipgloss.Style
	InactiveStyle lipgloss.Style
	TitleStyle    lipgloss.Style
}

func NewQuestionBox(title, message string, options []string, width int) QuestionBox {
	return QuestionBox{
		Title:         title,
		Message:       message,
		Options:       options,
		SelectedIdx:   0,
		Width:         width,
		Style:         theme.QuestionBoxStyle,
		ActiveStyle:   theme.QuestionBoxActiveStyle,
		InactiveStyle: theme.QuestionBoxInactiveStyle,
		TitleStyle:    theme.QuestionBoxTitleStyle,
	}
}

func (q QuestionBox) View() string {
	var sb strings.Builder

	sb.WriteString(q.TitleStyle.Render(q.Title))
	sb.WriteString("\n\n")
	sb.WriteString(theme.QuestionBoxMessageStyle.Render(q.Message))
	sb.WriteString("\n\n")

	for i, opt := range q.Options {
		if i == q.SelectedIdx {
			prefix := "▶ "
			sb.WriteString(q.ActiveStyle.Render(fmt.Sprintf("%s%d. %s", prefix, i+1, opt)))
		} else {
			prefix := "  "
			sb.WriteString(q.InactiveStyle.Render(fmt.Sprintf("%s%d. %s", prefix, i+1, opt)))
		}
		sb.WriteString("\n")
	}

	content := sb.String()
	return q.Style.Width(q.Width).Render(content)
}

func (q *QuestionBox) Up() {
	if q.SelectedIdx > 0 {
		q.SelectedIdx--
	}
}

func (q *QuestionBox) Down() {
	if q.SelectedIdx < len(q.Options)-1 {
		q.SelectedIdx++
	}
}

func (q QuestionBox) Selected() int {
	return q.SelectedIdx
}
