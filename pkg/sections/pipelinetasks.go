package sections

import (
	"azdoext/pkg/azdo"
	"azdoext/pkg/listitems"
	"azdoext/pkg/logger"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"
	"azdoext/pkg/utils"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type PipelineTasksSection struct {
	spinnerView       *string
	monitoredRunId    int
	logger            *logger.Logger
	tasklist          list.Model
	hidden            bool
	focused           bool
	ctx               context.Context
	cancelCtx         context.CancelFunc
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
	on := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Green)).SetString("on") // Green
	off := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Red)).SetString("off") // Red
	follow := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.White)).SetString("follow: ")
	followText := follow.String()
	if p.followRun {
		followText += on.String()
		return lipgloss.NewStyle().PaddingRight(1).Render(followText)
	}
	return followText + off.String()
}

func (p *PipelineTasksSection) View() string {
	title := styles.TitleStyle.Render(p.tasklist.Title)
	tasklistWidth := p.tasklist.Width()
	followViewStyle := lipgloss.NewStyle().PaddingLeft(tasklistWidth - p.tasklist.Paginator.TotalPages - lipgloss.Width(p.followView()))
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
	case teamsg.PipelineRunStateMsg:
		setitemscmd := p.tasklist.SetItems(msg.Items)
		tasks, listupdatecmd := p.tasklist.Update(msg)
		p.tasklist = tasks
		cmds = append(cmds, p.getRunState(p.ctx, p.monitoredRunId, 2*time.Second), listupdatecmd, setitemscmd)
		return p, tea.Batch(cmds...)
	case teamsg.PipelineRunIdMsg:
		if msg.RunId == p.monitoredRunId {
			return p, nil
		}
		setEmptyListCmd := p.tasklist.SetItems([]list.Item{})
		p.tasklist.Title = fmt.Sprint(msg.PipelineName)
		p.monitoredRunId = msg.RunId
		return p, tea.Batch(p.getRunState(p.ctx, msg.RunId, 0), p.spinner.Tick, setEmptyListCmd)
	case teamsg.LogMsg:
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
			if record.(listitems.PipelineRecordItem).Id == msg.StepRecordId {
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
						return teamsg.RecordSelectedMsg{
							RecordId: selectedRecord.Id,
						}
					})
			}
		}
	}
	return p, tea.Batch(cmds...)
}

func (p *PipelineTasksSection) getRunState(ctx context.Context, runId int, wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		p.logger.LogToFile("debug", fmt.Sprintf("fetching run state for runId %d after %s", runId, wait))
		err := utils.SleepWithContext(ctx, wait)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				p.logger.LogToFile("debug", "context cancelled while sleeping")
				return nil
			}
			panic(err)
		}
		records, err := p.buildclient.GetFilteredBuildTimelineRecords(ctx, build.GetBuildTimelineArgs{
			BuildId: &runId,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				p.logger.LogToFile("debug", "context cancelled while fetching timeline records")
				return nil
			}
			p.logger.LogToFile("error", fmt.Sprintf("error while fetching build timeline: %s", err))
			return nil
		}
		if len(records) == 0 {
			return nil
		}
		sortedRecords := sortRecords(records)
		sortedRecordItems := convertToItems(sortedRecords)
		sortedRecordItems = filterRecords(sortedRecordItems)
		items := p.addSymbol(sortedRecordItems)
		build, err := p.buildclient.GetBuilds(ctx, build.GetBuildsArgs{
			BuildIds: &[]int{runId},
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			panic(fmt.Sprintf("error getting build: %v", err))
		}
		buildstatus := build[0].Status
		return teamsg.PipelineRunStateMsg{
			Items:  items,
			Status: string(*buildstatus),
		}
	}
}

func getResultFromRecord(record build.TimelineRecord) build.TaskResult {
	if record.Result == nil {
		return ""
	}
	return *record.Result
}

func (p *PipelineTasksSection) getSymbol(status string) *string {
	if status == "inProgress" {
		return p.spinnerView
	} else {
		symbol := styles.SymbolMap[status].String()
		return &symbol
	}
}

func (p *PipelineTasksSection) addSymbol(records []listitems.PipelineRecordItem) []list.Item {
	var items []list.Item
	for _, item := range records {
		record := item
		record.Symbol = p.getSymbol(utils.StatusOrResult(&record.State, &record.Result))
		items = append(items, record)
	}
	return items
}

func (p *PipelineTasksSection) SetDimensions(width, height int) {
	p.tasklist.SetWidth(styles.DefaultSectionWidth)
	// here we decrease height by 2 to make room for the paginator view and the title
	p.tasklist.SetHeight(height - 2)
}

func buildPipelineRecordItemFromRecord(record build.TimelineRecord) listitems.PipelineRecordItem {
	recordStartTime := time.Time{}
	if record.StartTime != nil {
		recordStartTime = record.StartTime.Time
	}

	return listitems.PipelineRecordItem{
		StartTime: recordStartTime,
		Type:      *record.Type,
		Name:      *record.Name,
		State:     *record.State,
		Result:    getResultFromRecord(record),
		Id:        *record.Id,
	}
}

// We use this filter function to remove Phase records since they seem to have a 1-1 relationship to jobs, and also filter out pending tasks
func filterRecords(records []listitems.PipelineRecordItem) []listitems.PipelineRecordItem {
	return slices.DeleteFunc(records, func(record listitems.PipelineRecordItem) bool {
		return record.Type == "Phase" || record.State == build.TimelineRecordStateValues.Pending
	})
}

type RecordNode struct {
	Record   build.TimelineRecord
	Children []*RecordNode
}

type RootNode struct {
	Roots []*RecordNode
}

func (r RootNode) TimelineRecords() []build.TimelineRecord {
	var toTimelineRecords func([]build.TimelineRecord, []*RecordNode) []build.TimelineRecord
	toTimelineRecords = func(records []build.TimelineRecord, nodes []*RecordNode) []build.TimelineRecord {
		for _, node := range nodes {
			records = append(records, node.Record)
			records = toTimelineRecords(records, node.Children)
		}
		return records
	}

	tlrecords := []build.TimelineRecord{}
	tlrecords = toTimelineRecords(tlrecords, r.Roots)
	return tlrecords
}

func sortRecordTreeByOrder(nodes []*RecordNode) {
	slices.SortFunc(nodes, func(n1, n2 *RecordNode) int {
		return cmp.Compare(*n1.Record.Order, *n2.Record.Order)
	})
	for _, node := range nodes {
		sortRecordTreeByOrder(node.Children)
	}
}

func printRecords(nodes []*RecordNode) {
	for _, node := range nodes {
		fmt.Println(strings.Join([]string{
			strconv.Itoa(*(*node).Record.Order),
			*(*node).Record.Type,
			strings.ReplaceAll(*(*node).Record.Name, " ", ""),
			(*(*node).Record.Id).String(),
		}, "_"))
		printRecords(node.Children)
	}
}

func (r *RootNode) Print() {
	rjson, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatalf("unable to marshal root node: %v", err)
	}
	fmt.Println(string(rjson))
}

func sortRecords(records []build.TimelineRecord) []build.TimelineRecord {
	// build a hashmap so each record is easily accessible
	recordtree := make(map[uuid.UUID]*RecordNode)
	for _, record := range records {
		recordtree[*record.Id] = &RecordNode{Record: record}
	}
	// build a tree so the hierarchy Stage->Phase->Job->Task is respected
	root := RootNode{}
	for _, record := range records {
		if record.ParentId != nil {
			node := recordtree[*record.ParentId]
			node.Children = append(node.Children, recordtree[*record.Id])
			recordtree[*record.ParentId] = node
		} else {
			root.Roots = append(root.Roots, recordtree[*record.Id])
		}
	}
	// The siblings are in a random order, so sort each Children slice
	sortRecordTreeByOrder(root.Roots)

	sorted := root.TimelineRecords()

	return sorted
}

func convertToItems(records []build.TimelineRecord) []listitems.PipelineRecordItem {
	recorditems := []listitems.PipelineRecordItem{}
	for _, record := range records {
		recorditems = append(recorditems, buildPipelineRecordItemFromRecord(record))
	}
	return recorditems
}
