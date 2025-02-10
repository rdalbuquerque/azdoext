package pages

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type PageName string

const (
	Git          PageName = "git"
	Pipelines    PageName = "pipelines"
	Help         PageName = "help"
	PipelineRun  PageName = "pipelineRun"
	PipelineList PageName = "pipelineList"
)

type Stack []PageInterface

func (s *Stack) Push(page PageInterface) {
	*s = append(*s, page)
}

func (s *Stack) Pop() PageInterface {
	if len(*s) == 0 {
		return nil
	}
	page := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return page
}

func (s *Stack) Peek() PageInterface {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}

func attachView(view string, sectionView string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left, view, "  ", sectionView)
}

type helpKeys struct{}

func (h helpKeys) FullHelp() [][]key.Binding {
	return nil
}

func (h helpKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "help page"),
		),
		key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch section"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "restart"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "previous page"),
		),
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("↑/k ↓/j navigate and", "↵ select on all lists"),
		),
	}
}
