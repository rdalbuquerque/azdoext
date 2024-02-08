package azdo

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type item struct {
	title, desc string
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := i.title

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m *model) SetTaskList(ps pipelineState) {
	itemsList := []list.Item{}
	if m.pipelineState.Stages != nil {
		for _, stage := range m.pipelineState.Stages {
			itemsList = append(itemsList, item{title: m.formatStatusView(stage.State, stage.Result, stage.Name, ""), desc: stage.Log})
			for _, job := range stage.Jobs {
				itemsList = append(itemsList, item{title: m.formatStatusView(job.State, job.Result, job.Name, "  "), desc: job.Log})
				for _, task := range job.Tasks {
					itemsList = append(itemsList, item{title: m.formatStatusView(task.State, task.Result, task.Name, "    "), desc: task.Log})
				}
			}
		}
	}
	m.taskList.SetItems(itemsList)
}
