package listitems

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("170"))
	stagedFileStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
)

type PipelineItem struct {
	Name   string
	Id     int
	RunId  int
	Status string
	Result string
	Symbol *string
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
	title := titleStyle.Render(i.Name)
	if i.Symbol != nil {
		str = fmt.Sprintf("%s %s", *i.Symbol, title)
	} else {
		str = title
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type StagedFileItem struct {
	RawStatus string
	Name      string
	Staged    bool
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

	str := i.RawStatus
	if i.Staged {
		str = stagedFileStyle.Render(str)
	}
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type OptionName string

type ChoiceItem struct {
	Option OptionName
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

	str := string(i.Option)
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type PipelineRecordItem struct {
	Name      string
	Order     int
	StartTime time.Time
	Type      string
	RecordId  uuid.UUID
	State     build.TimelineRecordState
	Result    build.TaskResult
	Symbol    *string
}

func (p PipelineRecordItem) FilterValue() string { return "" }

type PipelineRecordItemDelegate struct{}

func (p PipelineRecordItemDelegate) Height() int                             { return 1 }
func (p PipelineRecordItemDelegate) Spacing() int                            { return 0 }
func (p PipelineRecordItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (p PipelineRecordItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(PipelineRecordItem)
	if !ok {
		return
	}

	var spacing string
	switch i.Type {
	case "Stage":
		spacing = ""
	case "Job":
		spacing = "  "
	case "Task":
		spacing = "    "
	}

	name := i.Name
	symbol := *i.Symbol
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(spacing, symbol, name))
}

type HelpKeys struct {
	AdditionalShortHelpKeys func() []key.Binding
}

func (h HelpKeys) FullHelp() [][]key.Binding {
	return nil
}

func (h HelpKeys) ShortHelp() []key.Binding {
	if h.AdditionalShortHelpKeys != nil {
		keys := h.AdditionalShortHelpKeys()
		return keys
	}
	return nil
}
