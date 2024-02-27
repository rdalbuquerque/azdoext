package azdo

import (
	"azdoext/pkgs/listitems"

	"github.com/charmbracelet/bubbles/list"
)

func (m *Model) SetTaskList(ps pipelineState) {
	itemsList := []list.Item{}
	if m.PipelineState.Stages != nil {
		for _, stage := range m.PipelineState.Stages {
			itemsList = append(itemsList, listitems.PipelineItem{Title: m.formatStatusView(stage.Status, stage.Name, ""), Desc: stage.Log})
			for _, job := range stage.Jobs {
				itemsList = append(itemsList, listitems.PipelineItem{Title: m.formatStatusView(job.Status, job.Name, "  "), Desc: job.Log})
				for _, task := range job.Tasks {
					if task.Status != "pending" {
						itemsList = append(itemsList, listitems.PipelineItem{Title: m.formatStatusView(task.Status, task.Name, "    "), Desc: task.Log})
					}
				}
			}
		}
	}
	m.TaskList.SetItems(itemsList)
}

func (m *Model) SetPipelineList() {
	for i := range m.PipelineList.Items() {
		symbol := m.getSymbol(m.PipelineList.Items()[i].(listitems.PipelineItem).Status)
		m.PipelineList.Items()[i] = listitems.PipelineItem{Symbol: symbol, Title: m.PipelineList.Items()[i].(listitems.PipelineItem).Title, Status: m.PipelineList.Items()[i].(listitems.PipelineItem).Status, Desc: m.PipelineList.Items()[i].(listitems.PipelineItem).Desc}
	}
}
