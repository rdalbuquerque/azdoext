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

type PipelineItem struct {
	Title  string
	Desc   any
	Status string
	Symbol string
}

func (i PipelineItem) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(PipelineItem)
	if !ok {
		return
	}

	var str string
	if i.Symbol != "" {
		str = fmt.Sprintf("%s %s", i.Symbol, i.Title)
	} else {
		str = i.Title
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m *Model) SetTaskList(ps pipelineState) {
	itemsList := []list.Item{}
	if m.PipelineState.Stages != nil {
		for _, stage := range m.PipelineState.Stages {
			itemsList = append(itemsList, PipelineItem{Title: m.formatStatusView(stage.Status, stage.Name, ""), Desc: stage.Log})
			for _, job := range stage.Jobs {
				itemsList = append(itemsList, PipelineItem{Title: m.formatStatusView(job.Status, job.Name, "  "), Desc: job.Log})
				for _, task := range job.Tasks {
					itemsList = append(itemsList, PipelineItem{Title: m.formatStatusView(task.Status, task.Name, "    "), Desc: task.Log})
				}
			}
		}
	}
	m.TaskList.SetItems(itemsList)
}

func (m *Model) SetPipelineList() {
	for i := range m.PipelineList.Items() {
		symbol := m.getSymbol(m.PipelineList.Items()[i].(PipelineItem).Status)
		m.PipelineList.Items()[i] = PipelineItem{Symbol: symbol, Title: m.PipelineList.Items()[i].(PipelineItem).Title, Status: m.PipelineList.Items()[i].(PipelineItem).Status, Desc: m.PipelineList.Items()[i].(PipelineItem).Desc}
	}
}
