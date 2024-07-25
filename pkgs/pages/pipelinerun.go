package pages

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"context"
	"fmt"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PipelineRunPage struct {
	current         bool
	name            PageName
	ctx             context.Context
	sections        map[sections.SectionName]sections.Section
	orderedSections []sections.SectionName
	shorthelp       string
	logger          *logger.Logger
}

func (p *PipelineRunPage) IsCurrentPage() bool {
	return p.current
}

func (p *PipelineRunPage) SetAsCurrentPage() {
	p.current = true
}

func (p *PipelineRunPage) UnsetCurrentPage() {
	p.current = false
}

func (p *PipelineRunPage) AddSection(section sections.Section) {
	secid := section.GetSectionIdentifier()
	if secid == "" {
		panic("section identifier is empty")
	}
	if p.sections == nil {
		p.sections = make(map[sections.SectionName]sections.Section)
	}
	if len(p.orderedSections) > 0 {
		for _, sec := range p.orderedSections {
			p.sections[sec].Blur()
		}
	}
	section.SetDimensions(0, styles.Height)
	section.Show()
	section.Focus()
	p.orderedSections = append(p.orderedSections, secid)
	p.sections[secid] = section
}

func NewPipelineRunPage(ctx context.Context, buildclient azdo.BuildClientInterface, azdoconfig azdo.Config) PageInterface {
	logger := logger.NewLogger("pipelinerun.log")
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	pipelineRunPage := &PipelineRunPage{
		ctx:       ctx,
		name:      PipelineRun,
		shorthelp: helpstring,
		logger:    logger,
	}
	pipetaskssec := sections.NewPipelineTasks(ctx, sections.PipelineTasks, buildclient)
	pipelineRunPage.AddSection(pipetaskssec)
	logvpsec := sections.NewLogViewport(ctx, sections.LogViewport, azdoconfig)
	pipelineRunPage.AddSection(logvpsec)
	pipelineRunPage.sections[sections.LogViewport].Blur()
	pipelineRunPage.sections[sections.PipelineTasks].Focus()
	return pipelineRunPage
}

func (p *PipelineRunPage) GetPageName() PageName {
	return PipelineRun
}

func (p *PipelineRunPage) SetDimensions(width, height int) {
	p.logger.LogToFile("info", fmt.Sprintf("setting dimensions for PipelineRunPage: task section width: %d, logviewport width: %d, height: %d", float32(width)*0.7, float32(width)*0.3, height))
	p.sections[sections.PipelineTasks].SetDimensions(int(float32(width)*0.2), height)
	p.sections[sections.LogViewport].SetDimensions(int(float32(width)*0.8), height)
}

func (p *PipelineRunPage) updateSections(msg tea.Msg) (map[sections.SectionName]sections.Section, []tea.Cmd) {
	updatedSections := make(map[sections.SectionName]sections.Section)
	var cmds []tea.Cmd
	for _, section := range p.orderedSections {
		sec, cmd := p.sections[section].Update(msg)
		updatedSections[section] = sec
		cmds = append(cmds, cmd)
	}
	return updatedSections, cmds
}

func (p *PipelineRunPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			p.switchSection()
			return p, nil
		}
	}
	sections, sectioncmds := p.updateSections(msg)
	cmds = append(cmds, sectioncmds...)
	p.sections = sections
	return p, tea.Batch(cmds...)
}

func (p *PipelineRunPage) View() string {
	var view string
	for _, section := range p.orderedSections {
		if !p.sections[section].IsHidden() {
			view = attachView(view, p.sections[section].View())
		}
	}
	viewWithHelp := lipgloss.JoinVertical(lipgloss.Top, view, p.shorthelp)
	return viewWithHelp
}

func (p *PipelineRunPage) switchSection() {
	shownSections := []sections.SectionName{}
	for _, section := range p.orderedSections {
		if !p.sections[section].IsHidden() {
			shownSections = append(shownSections, section)
		}
	}
	for i, sec := range shownSections {
		section := p.sections[sec]
		if section.IsFocused() {
			section.Blur()
			nextKey := shownSections[0] // default to the first key
			if i+1 < len(shownSections) {
				nextKey = shownSections[i+1] // if there's a next key, use it
			}
			p.sections[nextKey].Focus()
			return
		}
	}
}
