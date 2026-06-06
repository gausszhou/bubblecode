package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type TodoStatus int

const (
	TodoPending TodoStatus = iota
	TodoInProgress
	TodoCompleted
)

type TodoItem struct {
	ID     string
	Title  string
	Status TodoStatus
}

type TodoList struct {
	Items         []TodoItem
	Title         string
	TitleStyle    lipgloss.Style
	PendingIcon   string
	ProgressIcon  string
	CompleteIcon  string
	PendingStyle  lipgloss.Style
	ProgressStyle lipgloss.Style
	CompleteStyle lipgloss.Style
}

func NewTodoList(title string) TodoList {
	return TodoList{
		Title:         title,
		TitleStyle:    theme.TodoTitleStyle,
		PendingIcon:   "○",
		ProgressIcon:  "◐",
		CompleteIcon:  "●",
		PendingStyle:  theme.TodoPendingStyle,
		ProgressStyle: theme.TodoProgressStyle,
		CompleteStyle: theme.TodoCompleteStyle,
	}
}

func (tl TodoList) View() string {
	var sb strings.Builder
	sb.WriteString(tl.TitleStyle.Render(tl.Title))
	sb.WriteString("\n")

	if len(tl.Items) == 0 {
		sb.WriteString(theme.TodoEmptyStyle.Render("No tasks"))
		return sb.String()
	}

	for _, item := range tl.Items {
		icon := tl.PendingIcon
		style := tl.PendingStyle
		switch item.Status {
		case TodoInProgress:
			icon = tl.ProgressIcon
			style = tl.ProgressStyle
		case TodoCompleted:
			icon = tl.CompleteIcon
			style = tl.CompleteStyle
		}
		sb.WriteString(style.Render(fmt.Sprintf("  %s %s", icon, item.Title)))
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (tl *TodoList) AddItem(item TodoItem) {
	tl.Items = append(tl.Items, item)
}

func (tl *TodoList) UpdateStatus(id string, status TodoStatus) {
	for i := range tl.Items {
		if tl.Items[i].ID == id {
			tl.Items[i].Status = status
			return
		}
	}
}
