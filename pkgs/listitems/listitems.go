package listitems

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	stagedFileStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
)

type PipelineItem struct {
	Title  string
	Desc   any
	Status string
	Symbol string
}

func (i PipelineItem) FilterValue() string { return "" }

type ItemDelegate struct{}

func (d ItemDelegate) Height() int                             { return 1 }
func (d ItemDelegate) Spacing() int                            { return 0 }
func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(PipelineItem)
	if !ok {
		return
	}

	var str string
	titleStyle := lipgloss.NewStyle().MaxWidth(40)
	title := titleStyle.Render(i.Title)
	if i.Symbol != "" {
		str = fmt.Sprintf("%s %s", i.Symbol, title)
	} else {
		str = title
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type StagedFileItem struct {
	Name   string
	Staged bool
}

func (i StagedFileItem) FilterValue() string { return "" }

type GitItemDelegate struct{}

func (d GitItemDelegate) Height() int                             { return 1 }
func (d GitItemDelegate) Spacing() int                            { return 0 }
func (d GitItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d GitItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(StagedFileItem)
	if !ok {
		return
	}

	str := i.Name
	if i.Staged {
		str = stagedFileStyle.Render(str)
	}
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type OptionName string

type ChoiceItem struct {
	Choice OptionName
}

func (i ChoiceItem) FilterValue() string { return "" }

type ChoiceItemDelegate struct{}

func (d ChoiceItemDelegate) Height() int                             { return 1 }
func (d ChoiceItemDelegate) Spacing() int                            { return 0 }
func (d ChoiceItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d ChoiceItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(ChoiceItem)
	if !ok {
		return
	}

	str := string(i.Choice)
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type HelpKeys struct {
	AdditionalShortHelpKeys func() []key.Binding
}

func (h HelpKeys) FullHelp() [][]key.Binding {
	return nil
}

func (h HelpKeys) ShortHelp() []key.Binding {
	keys := defaultKeys()
	if h.AdditionalShortHelpKeys == nil {
		return keys
	}
	extra := h.AdditionalShortHelpKeys()
	keys = append(keys, extra...)
	return keys
}

func defaultKeys() []key.Binding {
	defaultHelp := []key.Binding{}
	defaultHelp = append(defaultHelp, key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	))
	return defaultHelp
}
