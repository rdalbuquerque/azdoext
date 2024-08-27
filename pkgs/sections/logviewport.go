package sections

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/azdosignalr"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/searchableviewport"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// this map stores task logs with log id and log content
	buildLogs utils.Logs

	// channel to receive logs
	logsChan chan utils.LogMsg

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
	return &LogViewportSection{
		logger:            logger,
		logviewport:       vp,
		ctx:               ctx,
		logsChan:          logsChan,
		sectionIdentifier: secid,
		azdoConfig:        azdoconfig,
		signalrClient:     signalrClient,
		StyledHelpText:    styledHelpText,
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
		if p.currentRunId == msg.RunId {
			return p, nil
		}
		p.logviewport.SetContent("")
		p.currentRunId = msg.RunId
		if p.signalrClient.IsConnected {
			p.logger.LogToFile("INFO", "active connection exists... closing connection")
			err := p.signalrClient.Conn.Close()
			if err != nil {
				p.logger.LogToFile("ERROR", fmt.Sprintf("error closing connection: %v", err))
				panic(err)
			}
		}
		p.signalrClient.Connect()
		p.buildLogs = make(utils.Logs)
		go p.signalrClient.StartReceivingLoop(p.logsChan)
		p.signalrClient.SendMessage("builddetailhub", "WatchBuild", []interface{}{p.azdoConfig.ProjectId, msg.RunId})
		return p, waitForLogs(p.logsChan)
	case utils.LogMsg:
		currentLog, ok := p.buildLogs[msg.StepRecordId]
		maxDigits := len(fmt.Sprintf("%d", 100000))
		if !ok {
			p.buildLogs[msg.StepRecordId] = fmt.Sprintf("%*d %s", maxDigits, 1, msg.NewContent+"\n")
			return p, waitForLogs(p.logsChan)
		}
		currentLog += fmt.Sprintf("%*d %s", maxDigits, len(strings.Split(currentLog, "\n"))+1, msg.NewContent+"\n")
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

func (p *LogViewportSection) SetDimensions(width, height int) {
	if width == 0 {
		width = styles.Width - styles.DefaultSectionWidth
	}
	p.logger.LogToFile("INFO", fmt.Sprintf("setting dimensions for LogViewportSection: width: %d, height: %d", width, height))
	// height - 1 to make space for the help text
	p.logviewport.SetDimensions(width, height-1)
}

type ToggleMaximizeMsg struct{}
