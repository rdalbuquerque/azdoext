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

// Main update function.
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
		log2file("WindowSizeMsg")
		m.height = msg.Height
		m.width = msg.Width
		for _, section := range m.sections {
			section.SetDimensions(msg.Width, msg.Height)
		}
		return m, nil
	case sections.GitPushedMsg:
		m.addSection(choice, sections.NewChoice)
	case sections.SubmitChoiceMsg:
		if msg == "Open PR" {
			m.addSection(openPR, sections.NewPRSection)
		}
	case azdo.PROpenedMsg:
		if msg {
			for _, section := range m.orderedSections {
				m.sections[section].Hide()
			}
			m.sections[azdoSection].Focus()
		}
	}
	for _, section := range m.orderedSections {
		log2file(fmt.Sprintf("section: %v", section))
		sec, cmd := m.sections[section].Update(msg)
		m.sections[section] = sec
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
	// switch msg := msg.(type) {
	// case InitializedMsg:
	// 	log2file("InitializedMsg")
	// 	return m, nil
	// case tea.WindowSizeMsg:
	// 	sections.ActiveStyle.Height(msg.Height - 2)
	// 	sections.InactiveStyle.Height(msg.Height - 2)
	// 	for _, section := range m.sections {
	// 		section.SetDimensions(msg.Width, msg.Height)
	// 	}
	// 	// m.azdo.SetHeights(msg.Height - 2)
	// 	m.prTextarea.SetHeight(msg.Height - 4)
	// 	m.prOrPipelineChoice.SetHeight(msg.Height - 2)
	// 	return m, nil
	// case tea.KeyMsg:
	// 	switch msg.Type {
	// 	case tea.KeyEnter:
	// 		log2file("Enter key pressed")
	// 		if m.activeSection == prOrPipelineSection {
	// 			if m.prOrPipelineChoice.SelectedItem().(listitems.StagedFileItem).Name == "Open PR" {
	// 				m.activeSection = openPRSection
	// 				m.prTextarea.Focus()
	// 				return m, nil
	// 			} else {
	// 				m.activeSection = azdoSection
	// 				return m, m.azdo.FetchPipelines(0)
	// 			}
	// 		}
	// 		if m.activeSection == openPRSection {
	// 			prtext, prcmd := m.prTextarea.Update(msg)
	// 			m.prTextarea = prtext
	// 			return m, prcmd
	// 		}
	// 		return m, tea.Batch(cmds...)
	// 	case tea.KeyCtrlC:
	// 		return m, tea.Quit
	// 	case tea.KeyTab:
	// 		m.switchSection()
	// 		// case tea.KeyCtrlS:
	// 		// 	if m.activeSection == openPRSection {
	// 		// 		titleAndDescription := strings.SplitN(m.prTextarea.Value(), "\n", 2)
	// 		// 		title := titleAndDescription[0]
	// 		// 		description := titleAndDescription[1]
	// 		// 		m.prTextarea.Blur()
	// 		// 		return m, tea.Batch(func() tea.Msg {
	// 		// 			return m.azdo.OpenPR(strings.Split(m.azdo.Branch, "/")[2], "master", title, description)
	// 		// 		}, m.azdo.FetchPipelines(0))
	// 		// 	}
	// 	}
	// case sections.GitPushedMsg:
	// 	if msg {
	// 		m.activeSection = prOrPipelineSection
	// 		return m, nil
	// 	}
	// case sections.GitPushingMsg:
	// case azdo.PipelinesFetchedMsg, azdo.PipelineIdMsg, azdo.PipelineStateMsg, azdo.PRMsg:
	// 	if m.activeSection == openPRSection {
	// 		m.activeSection = azdoSection
	// 	}
	// 	azdo, cmd := m.azdo.Update(msg)
	// 	m.azdo = azdo
	// 	return m, cmd
	// }
	// if m.activeSection == prOrPipelineSection {
	// 	prOrPipelineChoice, cmd := m.prOrPipelineChoice.Update(msg)
	// 	m.prOrPipelineChoice = prOrPipelineChoice
	// 	return m, cmd
	// }
	// if m.activeSection == openPRSection {
	// 	textarea, txtcmd := m.prTextarea.Update(msg)
	// 	m.prTextarea = textarea
	// 	return m, txtcmd
	// }
	// if m.activeSection == azdoSection {
	// 	azdo, cmd := m.azdo.Update(msg)
	// 	m.azdo = azdo
	// 	return m, cmd
	// }
	// return m, tea.Batch(cmds...)
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
