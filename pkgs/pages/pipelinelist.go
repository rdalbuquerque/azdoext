package pages

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"context"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PipelineListPage struct {
	selectedPipeline listitems.PipelineItem
	logger           *logger.Logger
	current          bool
	name             PageName
	ctx              context.Context
	sections         map[sections.SectionName]sections.Section
	orderedSections  []sections.SectionName
	shorthelp        string
}

func (p *PipelineListPage) IsCurrentPage() bool {
	return p.current
}

func (p *PipelineListPage) SetAsCurrentPage() {
	p.current = true
}

func (p *PipelineListPage) UnsetCurrentPage() {
	p.current = false
}

func (p *PipelineListPage) hasSection(section sections.SectionName) bool {
	_, ok := p.sections[section]
	return ok
}

func (p *PipelineListPage) AddSection(ctx context.Context, section sections.SectionName) {
	if !p.hasSection(section) {
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
}

func NewPipelineListPage(ctx context.Context) PageInterface {
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	logger := logger.NewLogger("pipelinelistpage.log")
	return &PipelineListPage{
		logger:    logger,
		ctx:       ctx,
		name:      PipelineList,
		shorthelp: helpstring,
	}
}

func (p *PipelineListPage) GetPageName() PageName {
	return PipelineList
}

func (p *PipelineListPage) SetDimensions(width, height int) {
	for s := range p.sections {
		p.sections[s].SetDimensions(width, height)
	}
}

func (p *PipelineListPage) updateSections(msg tea.Msg) (map[sections.SectionName]sections.Section, []tea.Cmd) {
	updatedSections := make(map[sections.SectionName]sections.Section)
	var cmds []tea.Cmd
	for _, section := range p.orderedSections {
		sec, cmd := p.sections[section].Update(msg)
		updatedSections[section] = sec
		cmds = append(cmds, cmd)
	}
	return updatedSections, cmds
}

func (p *PipelineListPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.current {
			switch msg.String() {
			case "tab":
				p.switchSection()
				return p, nil
			}
			sections, cmds := p.updateSections(msg)
			p.sections = sections
			return p, tea.Batch(cmds...)
		}
		return p, nil
	case sections.SubmitChoiceMsg:
		if choiseSec, ok := p.sections[sections.PipelineActionChoice]; !ok || !choiseSec.IsFocused() {
			return p, nil
		}
	case sections.GitInfoMsg:
		p.AddSection(p.ctx, sections.PipelineList)
	case sections.PipelineSelectedMsg:
		p.selectedPipeline = listitems.PipelineItem(msg)
		p.AddSection(p.ctx, sections.PipelineActionChoice)
		options := []list.Item{
			listitems.ChoiceItem{Option: sections.Options.GoToTasks},
			listitems.ChoiceItem{Option: sections.Options.RunPipeline},
		}
		sec, cmd := p.sections[sections.PipelineActionChoice].Update(sections.OptionsMsg(options))
		cmds = append(cmds, cmd)
		p.sections[sections.PrOrPipelineChoice] = sec
	}
	sections, sectioncmds := p.updateSections(msg)
	cmds = append(cmds, sectioncmds...)
	p.sections = sections
	return p, tea.Batch(cmds...)
}

func (p *PipelineListPage) View() string {
	var view string
	for _, section := range p.orderedSections {
		if !p.sections[section].IsHidden() {
			view = attachView(view, p.sections[section].View())
		}
	}
	viewWithHelp := lipgloss.JoinVertical(lipgloss.Top, view, p.shorthelp)
	return viewWithHelp
}

func (p *PipelineListPage) switchSection() {
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
