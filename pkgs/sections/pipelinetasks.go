package sections

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type PipelineRunIdMsg struct {
	RunId        int
	PipelineName string
}

type PipelineRunStateMsg []list.Item

type PipelineTasksSection struct {
	spinnerView    *string
	monitoredRunId int
	project        string
	logger         *logger.Logger
	tasklist       list.Model
	hidden         bool
	focused        bool
	ctx            context.Context
	spinner        spinner.Model
	buildclient    build.Client
}

func NewPipelineTasks(ctx context.Context) Section {
	logger := logger.NewLogger("pipelinetasks.log")
	tasklist := list.New([]list.Item{}, listitems.PipelineRecordItemDelegate{}, 40, 0)
	tasklist.SetShowStatusBar(false)
	tasklist.SetShowHelp(false)
	tasklist.SetShowPagination(false)

	spner := spinner.New()
	spner.Spinner = spinner.Dot
	spner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a9ff"))

	return &PipelineTasksSection{
		logger:      logger,
		tasklist:    tasklist,
		ctx:         ctx,
		spinner:     spner,
		spinnerView: utils.Ptr(spner.View()),
	}
}

func (p *PipelineTasksSection) IsHidden() bool {
	return p.hidden
}

func (p *PipelineTasksSection) IsFocused() bool {
	return p.focused
}

func (p *PipelineTasksSection) Hide() {
	p.hidden = true
}

func (p *PipelineTasksSection) Show() {
	p.hidden = false
}

func (p *PipelineTasksSection) Focus() {
	p.Show()
	p.focused = true
}

func (p *PipelineTasksSection) Blur() {
	p.focused = false
}

func (p *PipelineTasksSection) View() string {
	tasklistView := lipgloss.JoinVertical(lipgloss.Top, p.tasklist.View(), p.tasklist.Paginator.View())
	if p.focused {
		return styles.ActiveStyle.Render(tasklistView)
	}
	return styles.InactiveStyle.Render(tasklistView)
}

func (p *PipelineTasksSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case GitInfoMsg:
		azdoInfo := utils.ExtractAzdoInfo(msg.RemoteUrl)
		azdoconn := azuredevops.NewPatConnection(azdoInfo.OrgUrl, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
		buildclient, err := build.NewClient(p.ctx, azdoconn)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return p, nil
			}
			panic(err)
		}
		p.buildclient, p.project = buildclient, azdoInfo.Project
		return p, nil
	case PipelineRunStateMsg:
		p.logger.LogToFile("info", "received run state")
		setitemscmd := p.tasklist.SetItems(msg)
		tasks, listupdatecmd := p.tasklist.Update(msg)
		p.tasklist = tasks
		cmds = append(cmds, p.getRunState(p.ctx, p.monitoredRunId, 2*time.Second), listupdatecmd, setitemscmd)
		return p, tea.Batch(cmds...)
	case PipelineRunIdMsg:
		if msg.RunId == p.monitoredRunId {
			return p, nil
		}
		p.tasklist.Title = msg.PipelineName
		p.logger.LogToFile("info", fmt.Sprintf("received run id: %d", msg.RunId))
		p.monitoredRunId = msg.RunId
		return p, tea.Batch(p.getRunState(p.ctx, msg.RunId, 0), p.spinner.Tick)
	case spinner.TickMsg:
		spinner, cmd := p.spinner.Update(msg)
		p.spinner = spinner
		*p.spinnerView = spinner.View()
		return p, cmd
	case tea.KeyMsg:
		if p.focused {
			tasks, cmd := p.tasklist.Update(msg)
			cmds = append(cmds, cmd)
			p.tasklist = tasks

			if selectedRecord, ok := p.tasklist.SelectedItem().(listitems.PipelineRecordItem); ok {
				cmds = append(cmds,
					func() tea.Msg {
						return LogIdMsg{
							LogId:       selectedRecord.LogId,
							RecordState: selectedRecord.State,
							BuildId:     p.monitoredRunId,
						}
					})
			}
		}
	}
	return p, tea.Batch(cmds...)
}

func (p *PipelineTasksSection) getRunState(ctx context.Context, runId int, wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		p.logger.LogToFile("info", fmt.Sprintf("fetching builds of project %s and run id %d...", p.project, runId))
		err := utils.SleepWithContext(ctx, wait)
		if err != nil {
			p.logger.LogToFile("error", fmt.Sprintf("error while waiting: %s", err))
			return nil
		}
		timeline, err := p.buildclient.GetBuildTimeline(ctx, build.GetBuildTimelineArgs{
			Project: &p.project,
			BuildId: &runId,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			p.logger.LogToFile("error", fmt.Sprintf("error while fetching build timeline: %s", err))
			return nil
		}
		records := *timeline.Records
		p.logger.LogToFile("info", fmt.Sprintf("fetched %d records", len(records)))
		if len(records) == 0 {
			return nil
		}
		// filteredRecords := filterRecords(records)
		nodes := getRecordTree(records)
		pipelineRecords := p.getTaskList(nodes)
		sortedPipelineRecords := sortRecords(pipelineRecords)
		return PipelineRunStateMsg(sortedPipelineRecords)
	}
}

func sortRecords(records []list.Item) []list.Item {
	sort.Slice(records, func(i, j int) bool {
		if records[i].(listitems.PipelineRecordItem).StartTime.Equal(records[j].(listitems.PipelineRecordItem).StartTime) {
			// If start times are equal, ensure that "Job" comes after "Stage"
			if records[i].(listitems.PipelineRecordItem).Type == "Stage" && records[j].(listitems.PipelineRecordItem).Type == "Job" {
				return true
			}
			if records[i].(listitems.PipelineRecordItem).Type == "Job" && records[j].(listitems.PipelineRecordItem).Type == "Stage" {
				return false
			}
		}
		return records[i].(listitems.PipelineRecordItem).StartTime.Before(records[j].(listitems.PipelineRecordItem).StartTime)
	})
	return records

}

type recordNode struct {
	Record   build.TimelineRecord
	Children []*recordNode
}

func getRecordTree(records []build.TimelineRecord) []*recordNode {
	// Map records by ID and initialize children slice
	recordMap := make(map[uuid.UUID]*recordNode)
	for i, _ := range records {
		mapId := *records[i].Id
		recordMap[mapId] = &recordNode{Record: records[i]}
	}

	rootNodes := []*recordNode{}
	for i, _ := range records {
		record := records[i]
		if record.ParentId == nil {
			rootNodes = append(rootNodes, recordMap[*record.Id])
			continue
		}
		parent := recordMap[*record.ParentId]
		parent.Children = append(parent.Children, recordMap[*record.Id])
	}

	return rootNodes
}

func (p *PipelineTasksSection) getTaskList(node []*recordNode) []list.Item {
	var items []list.Item
	for _, node := range node {
		items = append(items, p.processNode(node)...)
	}
	return items
}

func (p *PipelineTasksSection) processNode(node *recordNode) []list.Item {
	var items []list.Item

	recordItem := p.buildPipelineRecordItem(node)
	if recordItem.State != build.TimelineRecordStateValues.Pending && recordItem.Type != "Phase" && recordItem.Type != "Checkpoint" {
		items = append(items, recordItem)
	}

	for _, node := range node.Children {
		childItems := p.processNode(node)
		items = append(items, childItems...)
	}

	return items
}

func (p *PipelineTasksSection) buildPipelineRecordItem(node *recordNode) listitems.PipelineRecordItem {
	recordStartTime := time.Time{}
	if node.Record.StartTime != nil {
		recordStartTime = node.Record.StartTime.Time
	}

	return listitems.PipelineRecordItem{
		StartTime: recordStartTime,
		Type:      *node.Record.Type,
		Name:      *node.Record.Name,
		State:     *node.Record.State,
		Result:    getResult(node),
		Symbol:    p.getSymbol(utils.StatusOrResult(node.Record.State, node.Record.Result)),
		LogId:     getLogId(node),
	}
}

func getResult(node *recordNode) build.TaskResult {
	if node.Record.Result == nil {
		return ""
	}
	return *node.Record.Result
}

func getLogId(node *recordNode) *int {
	if node.Record.Log == nil {
		return nil
	}
	logId := *node.Record.Log.Id
	return &logId
}

func (p *PipelineTasksSection) getSymbol(status string) *string {
	if status == "inProgress" {
		return p.spinnerView
	} else {
		symbol := styles.SymbolMap[status].String()
		return &symbol
	}
}

func (p *PipelineTasksSection) SetDimensions(width, height int) {
	// here we decrease height by 1 to make room for the paginator view
	p.tasklist.SetHeight(height - 1)
}
