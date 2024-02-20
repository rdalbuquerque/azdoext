package main

import (
	"fmt"
	"os"

	"explore-bubbletea/pkgs/azdo"
	"explore-bubbletea/pkgs/sections"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type gitOutputMsg string
type gitErrorMsg string
type sectionName string

const (
	commit      sectionName = "commit"
	worktree    sectionName = "worktree"
	choice      sectionName = "choice"
	azdoSection sectionName = "azdoSection"
	openPR      sectionName = "openPR"
)

var (
	activeStyle   = azdo.ActiveStyle.Copy()
	inactiveStyle = azdo.InactiveStyle.Copy()
)

type model struct {
	sections        map[sectionName]sections.Section
	orderedSections []sectionName
	height          int
	width           int
}

type newSection func() sections.Section

func (m *model) addSection(section sectionName, new newSection) {
	log2file(fmt.Sprintf("addSection: %v", section))
	for _, sec := range m.orderedSections {
		m.sections[sec].Blur()
	}
	newSection := new()
	newSection.SetDimensions(0, m.height-2)
	newSection.Show()
	newSection.Focus()
	m.orderedSections = append(m.orderedSections, section)
	m.sections[section] = newSection
}

func initialModel() model {
	commitSection := sections.NewCommitSection()
	commitSection.Show()
	commitSection.Focus()
	return model{
		sections: map[sectionName]sections.Section{
			commit: commitSection,
		},
		orderedSections: []sectionName{commit},
	}
}

type InitializedMsg bool

func (m *model) Init() tea.Cmd {
	log2file("Init")
	m.addSection(worktree, sections.NewWorktreeSection)
	log2file("worktree added")
	azdosection := azdo.New()
	log2file("azdo added")
	azdosection.Hide()
	m.sections[azdoSection] = azdosection
	m.orderedSections = append(m.orderedSections, azdoSection)
	_, cmd := m.sections[worktree].Update(sections.BroadcastGitInfoMsg(true))
	m.sections[worktree].Blur()
	m.sections[commit].Focus()
	return cmd
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.switchSection()
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		sections.ActiveStyle.Height(m.height - 2)
		sections.InactiveStyle.Height(m.height - 2)
		for _, section := range m.sections {
			section.SetDimensions(msg.Width, msg.Height-2)
		}
		return m, nil
	case sections.GitPushedMsg:
		m.addSection(choice, sections.NewChoice)
	case sections.SubmitChoiceMsg:
		if msg == "Open PR" {
			m.addSection(openPR, sections.NewPRSection)
		} else {
			m.setExclusiveFocus(azdoSection)
			return m, func() tea.Msg { return azdo.GoToPipelinesMsg(true) }
		}
	case azdo.PROpenedMsg:
		if msg {
			m.setExclusiveFocus(azdoSection)
		}
	}
	for _, section := range m.orderedSections {
		log2file(fmt.Sprintf("section: %v", section))
		sec, cmd := m.sections[section].Update(msg)
		m.sections[section] = sec
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	var view string
	for _, section := range m.orderedSections {
		if !m.sections[section].IsHidden() {
			view = attachView(view, m.sections[section].View())
		}
	}
	return view
}

func attachView(view string, sectionView string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left, view, "  ", sectionView)
}

func main() {
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}

// log func logs to a file
func log2file(msg string) {
	f, err := os.OpenFile("main-log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(msg + "\n"); err != nil {
		fmt.Println(err)
	}
}

func (m *model) switchSection() {
	shownSections := []sectionName{}
	for _, section := range m.orderedSections {
		if !m.sections[section].IsHidden() {
			shownSections = append(shownSections, section)
		}
	}
	for i, sec := range shownSections {
		section := m.sections[sec]
		if section.IsFocused() {
			section.Blur()
			nextKey := shownSections[0] // default to the first key
			if i+1 < len(shownSections) {
				nextKey = shownSections[i+1] // if there's a next key, use it
			}
			m.sections[nextKey].Focus()
			return
		}
	}
}

func (m *model) setExclusiveFocus(section sectionName) {
	for _, sec := range m.orderedSections {
		m.sections[sec].Blur()
		m.sections[sec].Hide()
	}
	m.sections[section].Focus()
}
