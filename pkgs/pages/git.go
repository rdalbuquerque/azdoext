package pages

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"context"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GitPage struct {
	current         bool
	name            PageName
	sections        map[sections.SectionName]sections.Section
	orderedSections []sections.SectionName
	shortHelp       string
}

func (p *GitPage) IsCurrentPage() bool {
	return p.current
}

func (p *GitPage) SetAsCurrentPage() {
	p.current = true
}

func (p *GitPage) UnsetCurrentPage() {
	p.current = false
}

func (p *GitPage) AddSection(ctx context.Context, section sections.SectionName) {
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

func NewGitPage() PageInterface {
	hk := helpKeys{}
	helpstring := bubbleshelp.New().View(hk)
	gitPage := &GitPage{}
	gitPage.name = Git
	gitPage.shortHelp = helpstring
	gitPage.AddSection(context.Background(), sections.Commit)
	gitPage.AddSection(context.Background(), sections.Worktree)
	gitPage.AddSection(context.Background(), sections.PrOrPipelineChoice)
	gitPage.AddSection(context.Background(), sections.OpenPR)
	gitPage.sections[sections.Commit].Focus()
	gitPage.sections[sections.Worktree].Blur()
	gitPage.sections[sections.PrOrPipelineChoice].Hide()
	gitPage.sections[sections.OpenPR].Hide()
	return gitPage
}

func (p *GitPage) GetPageName() PageName {
	return p.name
}

func (p *GitPage) Update(msg tea.Msg) (PageInterface, tea.Cmd) {
	// process any msg only if this page is the current page
	if p.current {
		var cmds []tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				p.switchSection()
				return p, nil
			}
		case sections.GitPushedMsg:
			p.SetFocus(sections.PrOrPipelineChoice)
			options := []list.Item{
				listitems.ChoiceItem{Option: sections.Options.OpenPR},
				listitems.ChoiceItem{Option: sections.Options.GoToPipelines},
			}
			sec, cmd := p.sections[sections.PrOrPipelineChoice].Update(sections.OptionsMsg(options))
			cmds = append(cmds, cmd)
			p.sections[sections.PrOrPipelineChoice] = sec
		case sections.SubmitChoiceMsg:
			if string(msg) == string(sections.Options.OpenPR) {
				p.SetFocus(sections.OpenPR)
			}
		}
		for _, section := range p.orderedSections {
			sec, cmd := p.sections[section].Update(msg)
			p.sections[section] = sec
			cmds = append(cmds, cmd)
		}
		return p, tea.Batch(cmds...)
	}
	return p, nil
}

func (p *GitPage) SetFocus(section sections.SectionName) {
	for _, sec := range p.orderedSections {
		if sec == section {
			p.sections[sec].Focus()
		} else {
			p.sections[sec].Blur()
		}
	}
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

func (p *GitPage) switchSection() {
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
	for s := range p.sections {
		p.sections[s].SetDimensions(width, height)
	}
}
