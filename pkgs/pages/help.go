package pages

import (
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type HelpPage struct {
	current  bool
	name     PageName
	sections map[sections.SectionName]sections.Section
}

func NewHelpPage() PageInterface {
	p := &HelpPage{}
	p.name = Help
	helpsec := sections.NewHelpSection(sections.Help)
	p.AddSection(helpsec)
	return p
}

func (p *HelpPage) IsCurrentPage() bool {
	return p.current
}

func (p *HelpPage) SetAsCurrentPage() {
	p.current = true
}

func (p *HelpPage) UnsetCurrentPage() {
	p.current = false
}

func (p *HelpPage) GetPageName() PageName {
	return p.name
}

func (p *HelpPage) AddSection(section sections.Section) {
	if p.sections == nil {
		p.sections = make(map[sections.SectionName]sections.Section)
	}
	section.SetDimensions(0, styles.Height)
	section.Show()
	section.Focus()
	p.sections[section.GetSectionIdentifier()] = section
}

func (p *HelpPage) View() string {
	helpsec := p.sections[sections.Help].(*sections.HelpSection)
	return p.sections[sections.Help].View() + "\n" + helpsec.ViewsearchHelp
}

func (p *HelpPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	if p.current {
		sec, cmd := p.sections[sections.Help].Update(msg)
		p.sections[sections.Help] = sec
		return p, cmd
	}
	return p, nil
}

func (p *HelpPage) SetDimensions(width, height int) {
	p.sections[sections.Help].SetDimensions(width, height)
}
