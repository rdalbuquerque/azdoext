package pages

import (
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"fmt"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PipelinesPage struct {
	current         bool
	name            PageName
	sections        map[sections.SectionName]sections.Section
	orderedSections []sections.SectionName
	shortHelp       string
}

func (p *PipelinesPage) IsCurrentPage() bool {
	return p.current
}

func (p *PipelinesPage) SetAsCurrentPage() {
	p.current = true
}

func (p *PipelinesPage) UnsetCurrentPage() {
	p.current = false
}

func (p *PipelinesPage) AddSection(section sections.SectionName) {
	newSection := sectionNewFuncs[section]()
	newSection.SetDimensions(0, styles.Height)
	newSection.Show()
	newSection.Focus()
	p.orderedSections = append(p.orderedSections, section)
	p.sections[section] = newSection
}

func NewAzdoPage() PageInterface {
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	p := &PipelinesPage{}
	p.name = Pipelines
	p.shortHelp = helpstring
	p.sections = make(map[sections.SectionName]sections.Section)
	p.AddSection(sections.AzdoSection)
	return p
}

func (p *PipelinesPage) GetPageName() PageName {
	return p.name
}

func (p *PipelinesPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	f, err := tea.LogToFile("pipelines-update.txt", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("handling msg: %v\n", msg))

	switch msg := msg.(type) {
	// only handle key messages if this page is the current page
	case tea.KeyMsg:
		if p.current {
			sections, cmds := p.updateSections(msg)
			p.sections = sections
			return p, tea.Batch(cmds...)
		}
	default:
		sections, cmds := p.updateSections(msg)
		p.sections = sections
		return p, tea.Batch(cmds...)
	}
	return p, nil
}

// update all sections and return sections and cmds
func (p *PipelinesPage) updateSections(msg tea.Msg) (map[sections.SectionName]sections.Section, []tea.Cmd) {
	var cmds []tea.Cmd
	updatedSections := make(map[sections.SectionName]sections.Section)
	for _, section := range p.orderedSections {
		sec, cmd := p.sections[section].Update(msg)
		updatedSections[section] = sec
		cmds = append(cmds, cmd)
	}
	return updatedSections, cmds
}

func (p *PipelinesPage) View() string {
	var view string
	for _, section := range p.orderedSections {
		if !p.sections[section].IsHidden() {
			view = attachView(view, p.sections[section].View())
		}
	}
	viewWithHelp := lipgloss.JoinVertical(lipgloss.Top, view, p.shortHelp)
	return viewWithHelp
}

func (p *PipelinesPage) SetDimensions(width, height int) {
	for _, section := range p.sections {
		section.SetDimensions(width, height)
	}
}
