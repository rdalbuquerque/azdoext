package sections

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type PipelineRunIdMsg struct {
	RunId        int
	PipelineName string
	ProjectId    string
}

type RecordSelectedMsg struct {
	RecordId string
}

type PipelineRunStateMsg []list.Item

type PipelineTasksSection struct {
	spinnerView       *string
	monitoredRunId    int
	logger            *logger.Logger
	tasklist          list.Model
	hidden            bool
	focused           bool
	ctx               context.Context
	spinner           spinner.Model
	buildclient       azdo.BuildClientInterface
	followRun         bool
	result            string
	buildStatus       string
	sectionIdentifier SectionName
}

func NewPipelineTasks(ctx context.Context, secid SectionName, buildclient azdo.BuildClientInterface) Section {
	logger := logger.NewLogger("pipelinetasks.log")
	tasklist := list.New([]list.Item{}, listitems.PipelineRecordItemDelegate{}, 0, 0)
	tasklist.SetShowTitle(false)
	tasklist.SetShowStatusBar(false)
	tasklist.SetShowHelp(false)
	tasklist.SetShowPagination(false)

	spner := spinner.New()
	spner.Spinner = spinner.Dot
	spner.Style = styles.SpinnerStyle

	return &PipelineTasksSection{
		logger:            logger,
		tasklist:          tasklist,
		ctx:               ctx,
		spinner:           spner,
		spinnerView:       utils.Ptr(spner.View()),
		buildclient:       buildclient,
		sectionIdentifier: secid,
	}
}

func (p *PipelineTasksSection) GetSectionIdentifier() SectionName {
	return p.sectionIdentifier
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

func (p *PipelineTasksSection) followView() string {
	if p.followRun {
		return "follow: on"
	}
	return "follow: off"
}

func (p *PipelineTasksSection) View() string {
	title := styles.TitleStyle.Render(p.tasklist.Title)
	if p.buildStatus == "completed" && len(p.result) > 0 {
		title = lipgloss.JoinHorizontal(lipgloss.Left, title, " ", *p.getSymbol(p.result))
	}
	tasklistWidth := p.tasklist.Width()
	followViewStyle := lipgloss.NewStyle().PaddingLeft(tasklistWidth - p.tasklist.Paginator.TotalPages - len(p.followView()))
	bottomView := lipgloss.JoinHorizontal(lipgloss.Bottom, p.tasklist.Paginator.View(), followViewStyle.Render(p.followView()))
	secView := lipgloss.JoinVertical(lipgloss.Top, title, p.tasklist.View(), bottomView)
	if p.focused {
		return styles.ActiveStyle.MaxWidth(tasklistWidth).Render(secView)
	}
	return styles.InactiveStyle.MaxWidth(tasklistWidth).Render(secView)
}

func (p *PipelineTasksSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case PipelineRunStateMsg:
		setitemscmd := p.tasklist.SetItems(msg)
		tasks, listupdatecmd := p.tasklist.Update(msg)
		p.tasklist = tasks
		cmds = append(cmds, p.getRunState(p.ctx, p.monitoredRunId, 2*time.Second), listupdatecmd, setitemscmd)
		return p, tea.Batch(cmds...)
	case PipelineRunIdMsg:
		if msg.RunId == p.monitoredRunId {
			return p, nil
		}
		setEmptyListCmd := p.tasklist.SetItems([]list.Item{})
		p.tasklist.Title = fmt.Sprintf(msg.PipelineName)
		p.monitoredRunId = msg.RunId
		return p, tea.Batch(p.getRunState(p.ctx, msg.RunId, 0), p.spinner.Tick, setEmptyListCmd)
	case utils.LogMsg:
		if len(msg.BuildResult) > 0 {
			p.buildStatus = msg.BuildStatus
			p.result = msg.BuildResult
			return p, nil
		}
		if !p.followRun {
			return p, nil
		}
		currentIndex := p.tasklist.Index()
		var recordIndex int
		for i, record := range p.tasklist.Items() {
			if record.(listitems.PipelineRecordItem).RecordId == string(msg.StepRecordId) {
				recordIndex = i
				break
			}
		}
		// If the record is not found, do not change the selection
		if recordIndex == 0 {
			recordIndex = currentIndex
		}
		p.tasklist.Select(recordIndex)
		return p, nil
	case spinner.TickMsg:
		spinner, cmd := p.spinner.Update(msg)
		p.spinner = spinner
		*p.spinnerView = spinner.View()
		return p, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return p, nil
		case "f":
			p.followRun = !p.followRun
			return p, nil
		}
		if p.focused {
			tasks, cmd := p.tasklist.Update(msg)
			cmds = append(cmds, cmd)
			p.tasklist = tasks

			if selectedRecord, ok := p.tasklist.SelectedItem().(listitems.PipelineRecordItem); ok {
				cmds = append(cmds,
					func() tea.Msg {
						return RecordSelectedMsg{
							RecordId: selectedRecord.RecordId,
						}
					})
			}
		}
	}
	return p, tea.Batch(cmds...)
}

func (p *PipelineTasksSection) getRunState(ctx context.Context, runId int, wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		err := utils.SleepWithContext(ctx, wait)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			panic(err)
		}
		records, err := p.buildclient.GetBuildTimelineRecords(ctx, build.GetBuildTimelineArgs{
			BuildId: &runId,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			p.logger.LogToFile("error", fmt.Sprintf("error while fetching build timeline: %s", err))
			return nil
		}
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
	sort.SliceStable(records, func(i, j int) bool {
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
		RecordId:  node.Record.Id.String(),
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
	p.tasklist.SetWidth(styles.DefaultSectionWidth)
	// here we decrease height by 2 to make room for the paginator view and the title
	p.tasklist.SetHeight(height - 2)
}
