package pages

import (
	"explore-bubbletea/pkgs/sections"
	"explore-bubbletea/pkgs/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type HelpPage struct {
	name     PageName
	sections map[sections.SectionName]sections.Section
}

func NewHelpPage() PageInterface {
	p := &HelpPage{}
	p.name = Help
	p.AddSection(sections.HelpSection)
	return p
}

func (p *HelpPage) GetPageName() PageName {
	return p.name
}

func (p *HelpPage) AddSection(section sections.SectionName) {
	if p.sections == nil {
		p.sections = make(map[sections.SectionName]sections.Section)
	}
	newSection := sectionNewFuncs[section]()
	newSection.SetDimensions(0, styles.Height)
	newSection.Show()
	newSection.Focus()
	p.sections[section] = newSection
}

func (p *HelpPage) View() string {
	return p.sections[sections.HelpSection].View()
}

func (p *HelpPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	sec, cmd := p.sections[sections.HelpSection].Update(msg)
	p.sections[sections.HelpSection] = sec
	return p, cmd
}

func (p *HelpPage) SwitchSection() {}

func (p *HelpPage) SetDimensions(width, height int) {
	p.sections[sections.HelpSection].SetDimensions(width, height)
}
