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
)

type LogViewportSection struct {
	logviewport       *searchableviewport.Model
	logger            *logger.Logger
	hidden            bool
	focused           bool
	ctx               context.Context
	followRun         bool
	currentStep       utils.StepRecordId
	sectionIdentifier SectionName
	azdoConfig        azdo.Config

	// this map stores task logs with log id and log content
	buildLogs utils.Logs

	// channel to receive logs
	logsChan chan utils.LogMsg

	// websocket connection to get live logs
	wsConn *azdosignalr.SignalRConn
}

func NewLogViewport(ctx context.Context, secid SectionName, azdoconfig azdo.Config) Section {
	logger := logger.NewLogger("logviewport.log")
	logger.LogToFile("INFO", "logviewport section initialized")
	vp := searchableviewport.New(0, 0)

	wsConn, err := azdosignalr.NewSignalRConn(azdoconfig.OrgName, azdoconfig.AccoundId, azdoconfig.ProjectId)
	if err != nil {
		panic(fmt.Errorf("signalr connection failed: %v", err))
	}

	logger.LogToFile("INFO", "azdoconfig: "+fmt.Sprintf("%+v", azdoconfig))
	logsChan := make(chan utils.LogMsg, 100)
	return &LogViewportSection{
		logger:            logger,
		logviewport:       vp,
		ctx:               ctx,
		logsChan:          logsChan,
		sectionIdentifier: secid,
		azdoConfig:        azdoconfig,
		wsConn:            wsConn,
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
	if p.focused {
		return styles.ActiveStyle.Render(p.logviewport.View())
	}
	return styles.InactiveStyle.Render(p.logviewport.View())
}

func (p *LogViewportSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case PipelineRunIdMsg:
		p.buildLogs = make(utils.Logs)
		go p.wsConn.StartReceivingLoop(p.logsChan)
		p.wsConn.SendMessage("builddetailhub", "WatchBuild", []interface{}{p.azdoConfig.ProjectId, msg.RunId})
		return p, waitForLogs(p.logsChan)
	case utils.LogMsg:
		p.logger.LogToFile("INFO", fmt.Sprintf("msg from timeline record: %s and step record: %s msg: %s", msg.TimelineRecordId, msg.StepRecordId, msg.NewContent))
		currentLog, ok := p.buildLogs[msg.StepRecordId]
		maxDigits := len(fmt.Sprintf("%d", 100000))
		if !ok {
			p.buildLogs[msg.StepRecordId] = fmt.Sprintf("%*d %s", maxDigits, 1, msg.NewContent+"\n")
			return p, waitForLogs(p.logsChan)
		}
		currentLog += fmt.Sprintf("%*d %s", maxDigits, len(strings.Split(currentLog, "\n"))+1, msg.NewContent+"\n")
		p.buildLogs[msg.StepRecordId] = currentLog
		if p.currentStep == msg.StepRecordId {
			p.logviewport.SetContent(currentLog)
		}
		if p.followRun {
			p.currentStep = msg.StepRecordId
			p.logviewport.SetContent(currentLog)
			p.logviewport.GotoBottom()
		}
		if msg.BuildStatus == "completed" {
			err := p.wsConn.Conn.Close()
			if err != nil {
				p.logger.LogToFile("ERROR", fmt.Sprintf("error closing connection: %v", err))
			}
			return p, nil
		}
		return p, waitForLogs(p.logsChan)
	case RecordSelectedMsg:
		p.logviewport.SetContent(p.buildLogs[utils.StepRecordId(msg.RecordId)])
		p.logviewport.GotoBottom()
		p.currentStep = utils.StepRecordId(msg.RecordId)
		return p, nil
	case tea.KeyMsg:
		p.logger.LogToFile("INFO", fmt.Sprintf("key pressed: %v", msg.String()))
		if msg.String() == "f" {
			p.followRun = !p.followRun
			return p, nil
		}
		if p.focused {
			p.logger.LogToFile("INFO", fmt.Sprintf("key pressed: %v", msg.String()))
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
	p.logger.LogToFile("INFO", fmt.Sprintf("setting dimensions for LogViewportSection: width: %d, height: %d", width, height))
	p.logviewport.SetDimensions(width, height)
}
