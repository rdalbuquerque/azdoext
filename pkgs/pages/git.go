package pages

import (
	"explore-bubbletea/pkgs/sections"
	"explore-bubbletea/pkgs/styles"
	"fmt"

	bubbleshelp "github.com/charmbracelet/bubbles/help"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GitPage struct {
	name            PageName
	sections        map[sections.SectionName]sections.Section
	orderedSections []sections.SectionName
	shortHelp       string
}

func (p *GitPage) AddSection(section sections.SectionName) {
	f, err := tea.LogToFile("debugheight.txt", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if p.sections == nil {
		p.sections = make(map[sections.SectionName]sections.Section)
	}
	if len(p.orderedSections) > 0 {
		for _, sec := range p.orderedSections {
			p.sections[sec].Blur()
		}
	}
	newSection := sectionNewFuncs[section]()
	f.WriteString(fmt.Sprintf("adding section [%v] with height [%d]\n", section, 0))
	newSection.SetDimensions(0, styles.Height)
	newSection.Show()
	newSection.Focus()
	p.orderedSections = append(p.orderedSections, section)
	p.sections[section] = newSection
}

func NewGitPage() PageInterface {
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	gitPage := &GitPage{}
	gitPage.name = Git
	gitPage.shortHelp = helpstring
	gitPage.AddSection(sections.Commit)
	gitPage.AddSection(sections.Worktree)
	gitPage.sections[sections.Commit].Focus()
	gitPage.sections[sections.Worktree].Blur()
	return gitPage
}

func (p *GitPage) GetPageName() PageName {
	return p.name
}

func (p *GitPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			p.SwitchSection()
			return p, nil
		}
	case sections.GitPushedMsg:
		p.AddSection(sections.ChoiceSection)
	case sections.SubmitChoiceMsg:
		if msg == "Open PR" {
			p.AddSection(sections.OpenPR)
		}
	}
	var cmds []tea.Cmd
	for _, section := range p.orderedSections {
		sec, cmd := p.sections[section].Update(msg)
		p.sections[section] = sec
		cmds = append(cmds, cmd)
	}
	return p, tea.Batch(cmds...)
}

func (p *GitPage) View() string {
	var view string
	for _, section := range p.orderedSections {
		if !p.sections[section].IsHidden() {
			view = attachView(view, p.sections[section].View())
		}
	}
	viewWithHelp := lipgloss.JoinVertical(lipgloss.Top, view, p.shortHelp)
	return viewWithHelp
}

func (p *GitPage) SwitchSection() {
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

func (p *GitPage) SetDimensions(width, height int) {
	f, err := tea.LogToFile("debugheight.txt", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for s := range p.sections {
		f.WriteString(fmt.Sprintf("setting dimensions for section [%v] with height [%d]\n", s, height))
		p.sections[s].SetDimensions(width, height)
	}
}
