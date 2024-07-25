package sections

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/azdosignalr"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/searchableviewport"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type RecordLog struct {
	Content         string
	LastRecordState build.TimelineRecordState
}

// msg with task id so log can be fetched
type LogIdMsg struct {
	BuildId     int
	LogId       *int
	RecordState build.TimelineRecordState
}

type LogViewportSection struct {
	logviewport       *searchableviewport.Model
	logger            *logger.Logger
	hidden            bool
	focused           bool
	project           string
	ctx               context.Context
	followRun         bool
	currentStep       utils.StepRecordId
	sectionIdentifier SectionName


	// this map stores task logs with log id and log content
	buildLogs map[int]RecordLog
}

func NewLogViewport(ctx context.Context, secid SectionName, azdoconfig azdo.Config) Section {
	logger := logger.NewLogger("logviewport.log")
	logger.LogToFile("INFO", "logviewport section initialized")
	vp := searchableviewport.New(0, 0)

	wsConn, err := azdosignalr.NewSignalRConn(azdoconfig.OrgName, azdoconfig.ProjectId)
	if err != nil {
		panic(fmt.Errorf("signalr connection failed: %v", err))
	}

	logsChan := make(chan utils.LogMsg, 100)
	return &LogViewportSection{
		logger:            logger,
		logviewport:       vp,
		ctx:               ctx,
		wsConn:            wsConn,
		logsChan:          logsChan,
		sectionIdentifier: secid,
		project:           azdoconfig.ProjectId,
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
		p.logviewport.SetContent("")
		return p, nil
	case tea.KeyMsg:
		if p.focused {
			vp, cmd := p.logviewport.Update(msg)
			p.logviewport = vp
			return p, cmd
		}
	case LogIdMsg:
		p.logger.LogToFile("INFO", "received LogIdMsg")
		if msg.LogId == nil {
			return p, nil
		}
		p.logger.LogToFile("INFO", fmt.Sprintf("log id: %v", *msg.LogId))
		content, err := p.GetLog(msg)
		if err != nil {
			p.logger.LogToFile("ERROR", err.Error())
			return p, nil
		}
		p.buildLogs[*msg.LogId] = RecordLog{
			Content:         content,
			LastRecordState: msg.RecordState,
		}
		p.logviewport.SetContent(content)
		return p, nil
	case GitInfoMsg:
		p.logger.LogToFile("INFO", "received git info")
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
	}
	return p, nil
}

func (p *LogViewportSection) GetLog(msg LogIdMsg) (string, error) {
	logId := *msg.LogId
	if p.shouldGetLog(logId) {
		logReader, err := p.buildclient.GetBuildLog(p.ctx, build.GetBuildLogArgs{
			Project: &p.project,
			BuildId: &msg.BuildId,
			LogId:   &logId,
		})
		if err != nil {
			return "", fmt.Errorf("error fetching log: %v", err)
		}

func (p *LogViewportSection) SetDimensions(width, height int) {
	p.logviewport.SetDimensions(styles.Width, height)
}
