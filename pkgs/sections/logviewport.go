package sections

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/azdosignalr"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/teamsg"
	"azdoext/pkgs/utils"
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/rdalbuquerque/viewsearch"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/muesli/reflow/wordwrap"
)

type LogViewportSection struct {
	logviewport       *viewsearch.Model
	logger            *logger.Logger
	hidden            bool
	focused           bool
	ctx               context.Context
	StyledHelpText    string
	followRun         bool
	currentStep       uuid.UUID
	currentRunId      int
	sectionIdentifier SectionName
	azdoConfig        azdo.Config
	buildclient       azdo.BuildClientInterface
	buildStatus       string

	// this map stores task logs with log id and log content
	buildLogs utils.Logs

	// channel to receive logs
	logsChan chan teamsg.LogMsg

	// cancel function to stop receiving logs and close signalr connection
	readLogsCtx       context.Context
	cancelReceiveLogs context.CancelFunc

	// channel to signal that the connection is closed
	connClosedChan chan bool

	// channel to signal an error while closing the connection
	connClosedErrChan chan error

	// websocket connection to get live logs
	signalrClient *azdosignalr.SignalRClient
}

func NewLogViewport(ctx context.Context, secid SectionName, azdoconfig azdo.Config) Section {
	logger := logger.NewLogger("logviewport.log")
	vp := viewsearch.New()
	vp.SetShowHelp(false)

	styledHelpText := styles.ShortHelpStyle.Render("/ find • alt+m maximize")

	signalrClient := azdosignalr.NewSignalR(azdoconfig.OrgName, azdoconfig.AccoundId, azdoconfig.ProjectId)

	connClosedChan := make(chan bool)
	connClosedErrChan := make(chan error)

	buildclient := azdo.NewBuildClient(ctx, azdoconfig.OrgUrl, azdoconfig.ProjectId, azdoconfig.PAT)
	return &LogViewportSection{
		logger:            logger,
		logviewport:       &vp,
		ctx:               ctx,
		connClosedChan:    connClosedChan,
		connClosedErrChan: connClosedErrChan,
		sectionIdentifier: secid,
		azdoConfig:        azdoconfig,
		signalrClient:     signalrClient,
		StyledHelpText:    styledHelpText,
		buildclient:       buildclient,
	}
}

func (p *LogViewportSection) GetSectionIdentifier() SectionName {
	return p.sectionIdentifier
}

func (p *LogViewportSection) IsHidden() bool {
	return p.hidden
}

func (p *LogViewportSection) IsFocused() bool {
	return p.focused
}

func (p *LogViewportSection) Hide() {
	p.hidden = true
}

func (p *LogViewportSection) Show() {
	p.hidden = false
}

func (p *LogViewportSection) Focus() {
	p.Show()
	p.focused = true
}

func (p *LogViewportSection) Blur() {
	p.focused = false
}

func (p *LogViewportSection) View() string {
	helpPlacement := lipgloss.NewStyle().PaddingLeft(p.logviewport.Viewport.Width - len(p.StyledHelpText) + 18)
	logsAndHelp := lipgloss.JoinVertical(lipgloss.Top, p.logviewport.View(), helpPlacement.Render(p.StyledHelpText))
	if p.focused {
		return styles.ActiveStyle.Render(logsAndHelp)
	}
	return styles.InactiveStyle.Render(logsAndHelp)
}

func (p *LogViewportSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case teamsg.ToggleMaximizeMsg:
		wrappedContent := wordwrap.String(p.buildLogs[p.currentStep], p.logviewport.Viewport.Width)
		p.logviewport.SetContent(wrappedContent)
		return p, nil
	case teamsg.PipelineRunIdMsg:
		if p.currentRunId == msg.RunId {
			return p, nil
		}
		buildsLogs := make(utils.Logs)
		p.buildLogs = buildsLogs
		p.buildStatus = msg.Status
		p.currentRunId = msg.RunId
		p.cancelReceiveLogsIfExists()

		logsChan := make(chan teamsg.LogMsg, 100)
		p.logsChan = logsChan

		p.logviewport.SetContent("")
		if msg.Status == "completed" {
			p.handleCompletedRun(msg.RunId)
			return p, nil
		}
		ctx, cancel := context.WithCancel(p.ctx)
		p.readLogsCtx = ctx
		p.cancelReceiveLogs = cancel
		p.startMonitoringLogs(ctx, msg.RunId, p.connClosedChan, p.connClosedErrChan)
		return p, waitForLogs(p.logsChan)
	case teamsg.LogMsg:
		currentLog, ok := p.buildLogs[msg.StepRecordId]
		if !ok {
			p.buildLogs[msg.StepRecordId] = formatLine(msg.NewContent, 1)
			return p, waitForLogs(p.logsChan)
		}
		lineNum := len(strings.Split(currentLog, "\n")) + 1
		currentLog += formatLine(msg.NewContent, lineNum)
		p.buildLogs[msg.StepRecordId] = currentLog
		if p.currentStep == msg.StepRecordId {
			p.logviewport.SetContent(wordwrap.String(currentLog, p.logviewport.Viewport.Width))
		}
		if p.followRun {
			p.currentStep = msg.StepRecordId
			p.logviewport.SetContent(wordwrap.String(currentLog, p.logviewport.Viewport.Width))
			p.logviewport.GotoBottom()
		}
		return p, waitForLogs(p.logsChan)
	case teamsg.RecordSelectedMsg:
		wrappedContent := wordwrap.String(p.buildLogs[msg.RecordId], p.logviewport.Viewport.Width)
		p.logviewport.SetContent(wrappedContent)
		p.logviewport.GotoBottom()
		p.currentStep = msg.RecordId
		return p, nil
	case tea.KeyMsg:
		if msg.String() == "f" {
			p.followRun = !p.followRun
			return p, nil
		}
		if p.focused {
			vp, cmd := p.logviewport.Update(msg)
			p.logviewport = &vp
			return p, cmd
		}
	}
	return p, nil
}

func (p *LogViewportSection) startMonitoringLogs(ctx context.Context, runId int, connClosedChan chan bool, connClosedErrChan chan error) {
	p.logger.LogToFile("INFO", "connecting to signalr")
	err := p.signalrClient.Connect()
	if err != nil {
		panic(fmt.Sprintf("error connecting to signalr: %v", err))
	}
	go p.signalrClient.StartReceivingLoop(ctx, p.logsChan, connClosedChan, connClosedErrChan)
	p.signalrClient.SendWatchBuildMessage(runId)
}

func waitForLogs(logsChan chan teamsg.LogMsg) tea.Cmd {
	return func() tea.Msg {
		return <-logsChan
	}
}

func (p *LogViewportSection) cancelReceiveLogsIfExists() {
	if p.cancelReceiveLogs != nil {
		p.cancelReceiveLogs()
		p.cancelReceiveLogs = nil
		select {
		case <-p.connClosedChan:
			p.logger.LogToFile("INFO", "connection closed")
		case err := <-p.connClosedErrChan:
			p.logger.LogToFile("ERROR", fmt.Sprintf("error closing connection: %v", err))
			panic(fmt.Sprintf("error closing connection: %v", err))
		}
	}
}

func (p *LogViewportSection) handleCompletedRun(runId int) {
	records, err := p.buildclient.GetBuildTimelineRecords(p.ctx, build.GetBuildTimelineArgs{
		BuildId: &runId,
	})
	if err != nil {
		panic(fmt.Sprintf("error getting timeline records: %v", err))
	}
	for _, item := range records {
		recordId := *item.Id
		recordLogId := getLogId(item)
		if p.buildLogs[recordId] != "" || recordLogId == nil {
			continue
		}
		logreader, err := p.buildclient.GetTimelineRecordLog(p.ctx, build.GetBuildLogArgs{
			Project: &p.azdoConfig.ProjectId,
			BuildId: &p.currentRunId,
			LogId:   recordLogId,
		})
		if err != nil {
			panic(fmt.Sprintf("error getting log: %v", err))
		}
		p.buildLogs[recordId] = formatLog(logreader)
	}
}

func getLogId(item build.TimelineRecord) *int {
	if item.Log == nil {
		return nil
	}
	return item.Log.Id
}

func formatLine(line string, lineNum int) string {
	maxDigits := len(fmt.Sprintf("%d", 100000))
	line = removeTimestamp(line)
	formattedLine := fmt.Sprintf("%*d: %s\n", maxDigits, lineNum, line)
	return formattedLine
}

func removeTimestamp(line string) string {
	// Regular expression to match the timestamp pattern
	pattern := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\s+`
	re := regexp.MustCompile(pattern)
	// Replace the matched timestamp with an empty string
	return re.ReplaceAllString(line, "")
}

func formatLog(log io.ReadCloser) string {
	scanner := bufio.NewScanner(log)
	var formattedLog string
	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		newLine := formatLine(line, lineNum)
		formattedLog += newLine
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return formattedLog
}

func (p *LogViewportSection) SetDimensions(width, height int) {
	if width == 0 {
		width = styles.Width - styles.DefaultSectionWidth
	}
	// height - 1 to make space for the help text
	p.logviewport.SetDimensions(width, height-1)
}
