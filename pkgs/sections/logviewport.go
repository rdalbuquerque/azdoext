package sections

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/azdosignalr"
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/searchableviewport"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/muesli/reflow/wordwrap"
)

type LogViewportSection struct {
	logviewport       *searchableviewport.Model
	logger            *logger.Logger
	hidden            bool
	focused           bool
	ctx               context.Context
	StyledHelpText    string
	followRun         bool
	currentStep       utils.StepRecordId
	currentRunId      int
	sectionIdentifier SectionName
	azdoConfig        azdo.Config
	logclient         azdo.LogClientInterface

	// this map stores task logs with log id and log content
	buildLogs utils.Logs

	// channel to receive logs
	logsChan chan utils.LogMsg

	// cancel function to stop receiving logs and close signalr connection
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
	logger.LogToFile("INFO", "logviewport section initialized")
	vp := searchableviewport.New(0, 0)

	styledHelpText := styles.ShortHelpStyle.Render("/ find • alt+m maximize")

	signalrClient := azdosignalr.NewSignalR(azdoconfig.OrgName, azdoconfig.AccoundId, azdoconfig.ProjectId)

	logger.LogToFile("INFO", "azdoconfig: "+fmt.Sprintf("%+v", azdoconfig))

	logsChan := make(chan utils.LogMsg, 100)
	buildLogs := make(utils.Logs)
	connClosedChan := make(chan bool)
	connClosedErrChan := make(chan error)

	logclient := azdo.NewLogClient(ctx, azdoconfig.OrgUrl, azdoconfig.ProjectId, azdoconfig.PAT)
	return &LogViewportSection{
		logger:            logger,
		logviewport:       vp,
		ctx:               ctx,
		logsChan:          logsChan,
		connClosedChan:    connClosedChan,
		connClosedErrChan: connClosedErrChan,
		sectionIdentifier: secid,
		azdoConfig:        azdoconfig,
		signalrClient:     signalrClient,
		StyledHelpText:    styledHelpText,
		logclient:         logclient,
		buildLogs:         buildLogs,
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
	case ToggleMaximizeMsg:
		wrappedContent := wordwrap.String(p.buildLogs[p.currentStep], p.logviewport.Viewport.Width)
		p.logviewport.SetContent(wrappedContent)
		return p, nil
	case PipelineRunIdMsg:
		p.logviewport.SetContent("")
		p.logger.LogToFile("INFO", fmt.Sprintf("run id: %d, status: %s", msg.RunId, msg.Status))
		if p.currentRunId == msg.RunId || msg.Status == "completed" {
			p.currentRunId = msg.RunId
			p.logger.LogToFile("INFO", "run already selected or completed")
			return p, nil
		}
		p.currentRunId = msg.RunId
		if p.cancelReceiveLogs != nil {
			p.cancelReceiveLogs()
			select {
			case <-p.connClosedChan:
				p.logger.LogToFile("INFO", "connection closed")
			case <-p.connClosedChan:
				p.logger.LogToFile("ERROR", fmt.Sprintf("error closing connection: %v", <-p.connClosedErrChan))
				panic("error closing connection")
			}
		}

		recLoopCtx, cancel := context.WithCancel(p.ctx)
		p.cancelReceiveLogs = cancel
		p.signalrClient.Connect()
		go p.signalrClient.StartReceivingLoop(recLoopCtx, p.logsChan, p.connClosedChan, p.connClosedErrChan)
		p.signalrClient.SendMessage("builddetailhub", "WatchBuild", []interface{}{p.azdoConfig.ProjectId, msg.RunId})
		return p, waitForLogs(p.logsChan)
	case utils.LogMsg:
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
	case RecordSelectedMsg:
		wrappedContent := wordwrap.String(p.buildLogs[utils.StepRecordId(msg.RecordId)], p.logviewport.Viewport.Width)
		p.logviewport.SetContent(wrappedContent)
		p.logviewport.GotoBottom()
		p.currentStep = utils.StepRecordId(msg.RecordId)
		return p, nil
	case tea.KeyMsg:
		if msg.String() == "f" {
			p.followRun = !p.followRun
			return p, nil
		}
		if p.focused {
			vp, cmd := p.logviewport.Update(msg)
			p.logviewport = vp
			return p, cmd
		}
	}
	return p, nil
}

func waitForLogs(logsChan chan utils.LogMsg) tea.Cmd {
	return func() tea.Msg {
		return <-logsChan
	}
}

func (p *LogViewportSection) handleCompletedRun(msg []list.Item) {
	for _, item := range msg {
		recordId := utils.StepRecordId(item.(listitems.PipelineRecordItem).RecordId)
		recordLogId := item.(listitems.PipelineRecordItem).LogId
		if p.buildLogs[recordId] != "" || recordLogId == nil {
			continue
		}
		logreader, err := p.logclient.GetTimelineRecordLog(p.ctx, build.GetBuildLogArgs{
			Project: &p.azdoConfig.ProjectId,
			BuildId: &p.currentRunId,
			LogId:   recordLogId,
		})
		if err != nil {
			panic(fmt.Sprintf("error getting log: %v", err))
		}
		p.buildLogs[utils.StepRecordId(item.(listitems.PipelineRecordItem).RecordId)] = formatLog(logreader)
	}
}

func formatLine(line string, lineNum int) string {
	maxDigits := len(fmt.Sprintf("%d", 100000))
	line = removeTimestamp(line)
	formattedLine := fmt.Sprintf("%*d: %s\n", maxDigits, lineNum, line)
	return formattedLine
}

func removeTimestamp(line string) string {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return line // Return the original line if there is no timestamp
	}
	return parts[1]
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

type ToggleMaximizeMsg struct{}
