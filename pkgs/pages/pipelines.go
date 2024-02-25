package pages

import (
	"explore-bubbletea/pkgs/sections"
	"explore-bubbletea/pkgs/styles"
	"fmt"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PipelinesPage struct {
	name            PageName
	sections        map[sections.SectionName]sections.Section
	orderedSections []sections.SectionName
	shortHelp       string
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

	var cmds []tea.Cmd
	for _, section := range p.orderedSections {
		sec, cmd := p.sections[section].Update(msg)
		p.sections[section] = sec
		cmds = append(cmds, cmd)
	}
	return p, tea.Batch(cmds...)
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

func (p *PipelinesPage) SwitchSection() {
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

func (p *PipelinesPage) SetDimensions(width, height int) {
	for _, section := range p.sections {
		section.SetDimensions(width, height)
	}
}
