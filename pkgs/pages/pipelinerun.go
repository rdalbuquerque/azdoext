package pages

import (
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"context"

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

func (p *PipelineRunPage) AddSection(ctx context.Context, section sections.SectionName) {
	if p.sections == nil {
		p.sections = make(map[sections.SectionName]sections.Section)
	}
	if len(p.orderedSections) > 0 {
		for _, sec := range p.orderedSections {
			p.sections[sec].Blur()
		}
	}
	newSection := sectionNewFuncs[section](ctx)
	newSection.SetDimensions(0, styles.Height)
	newSection.Show()
	newSection.Focus()
	p.orderedSections = append(p.orderedSections, section)
	p.sections[section] = newSection
}

func NewPipelineRunPage(ctx context.Context) PageInterface {
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	pipelineRunPage := &PipelineRunPage{
		ctx:       ctx,
		name:      PipelineRun,
		shorthelp: helpstring,
	}
	pipelineRunPage.AddSection(ctx, sections.PipelineTasks)
	pipelineRunPage.AddSection(ctx, sections.LogViewport)
	pipelineRunPage.sections[sections.LogViewport].Blur()
	pipelineRunPage.sections[sections.PipelineTasks].Focus()
	return pipelineRunPage
}

func (p *PipelineRunPage) GetPageName() PageName {
	return PipelineRun
}

func (p *PipelineRunPage) SetDimensions(width, height int) {
	for s := range p.sections {
		p.sections[s].SetDimensions(width, height)
	}
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
