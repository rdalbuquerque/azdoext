package sections

import (
	"azdoext/pkg/azdo"
	"azdoext/pkg/listitems"
	"azdoext/pkg/logger"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"
	"azdoext/pkg/utils"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type LinkedList struct {
	head *Node
}

type Node struct {
	Record build.TimelineRecord
	Next   *Node
}

func (l *LinkedList) Insert(record build.TimelineRecord) {
	newNode := &Node{Record: record}
	// case where the linked list is brand new
	if l.head == nil {
		l.head = newNode
	} else {
		currentNode := l.head
		for currentNode != nil {
			// stage record handling (parentId = null)
			if record.ParentId == nil {
				// look for the previous stage and insert next to it
				if *currentNode.Record.Order == *record.Order-1 {
					newNode.Next = currentNode.Next
					currentNode.Next = newNode
					return
				}
				// if first stage, just assumes head
				if *record.Order == 1 {
					newNode.Next = l.head
					l.head = newNode
					return
				}
			} else {
				// case of non-stage typed record with order = 1 -> should look for it's parent and be inserted next to it
				if *record.Order == 1 && *record.ParentId == *currentNode.Record.Id {
					newNode.Next = currentNode.Next
					currentNode.Next = newNode
					return
				}
				// case of non-stage typed record with order > 1 -> should look for the previous order item of the same parent and be inserted next to it
				if currentNode.Record.ParentId != nil {
					if *currentNode.Record.ParentId == *record.ParentId && *currentNode.Record.Order == *record.Order-1 {
						newNode.Next = currentNode.Next
						currentNode.Next = newNode
						return
					}
					if currentNode.Next != nil {
						if *currentNode.Record.ParentId == *record.ParentId && (*currentNode.Next.Record.Type != *record.Type || *currentNode.Next.Record.Order > *record.Order) {
							newNode.Next = currentNode.Next
							currentNode.Next = newNode
							return
						}
					}
				}
				if currentNode.Next != nil {
					if *record.ParentId == *currentNode.Record.Id && (*currentNode.Next.Record.Type != *record.Type || *currentNode.Next.Record.Order > *record.Order) {
						newNode.Next = currentNode.Next
						currentNode.Next = newNode
						return
					}
				}
			}
			if currentNode.Next == nil {
				currentNode.Next = newNode
				return
			}
			// if it's none of these, go to the next one and keep looking
			currentNode = currentNode.Next
		}
	}
}

func (l LinkedList) Print() {
	currentNode := l.head
	for currentNode != nil {
		fmt.Printf("type: %s | order: %d | name: %s\n", *currentNode.Record.Type, *currentNode.Record.Order, *currentNode.Record.Name)
		currentNode = currentNode.Next
	}
}

func (l LinkedList) ToSliceOfItems() []listitems.PipelineRecordItem {
	currentNode := l.head
	itemlist := []listitems.PipelineRecordItem{}
	for currentNode != nil {
		itemlist = append(itemlist, buildPipelineRecordItem(*currentNode))
		currentNode = currentNode.Next
	}
	return itemlist
}

func getRecordLinkedList(records []build.TimelineRecord) LinkedList {
	stages := []build.TimelineRecord{}
	phases := []build.TimelineRecord{}
	jobs := []build.TimelineRecord{}
	tasks := []build.TimelineRecord{}
	for _, record := range records {
		switch *record.Type {
		case "Stage":
			stages = append(stages, record)
		case "Phase":
			phases = append(phases, record)
		case "Job":
			jobs = append(jobs, record)
		case "Task":
			tasks = append(tasks, record)
		}
	}
	ll := LinkedList{}
	for _, record := range stages {
		ll.Insert(record)
	}
	for _, record := range phases {
		ll.Insert(record)
	}
	for _, record := range jobs {
		ll.Insert(record)
	}
	for _, record := range tasks {
		ll.Insert(record)
	}
	return ll
}

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
		recordsll := getRecordLinkedList(records)
		sortedRecordItems := recordsll.ToSliceOfItems()
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

func getResult(node Node) build.TaskResult {
	if node.Record.Result == nil {
		return ""
	}
	return *node.Record.Result
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

func buildPipelineRecordItem(node Node) listitems.PipelineRecordItem {
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
		Id:        *node.Record.Id,
	}
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

func filterRecords(records []listitems.PipelineRecordItem) []listitems.PipelineRecordItem {
	return slices.DeleteFunc(records, func(record listitems.PipelineRecordItem) bool {
		return record.Type == "Phase" || record.State == build.TimelineRecordStateValues.Pending
	})
}
