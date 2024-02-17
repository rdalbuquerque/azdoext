package main

import (
	"fmt"
	"os"
	"strings"

	"net/url"

	"explore-bubbletea/pkgs/azdo"
	"explore-bubbletea/pkgs/listitems"
	"explore-bubbletea/pkgs/sections"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
)

type gitOutputMsg string
type gitErrorMsg string

var (
	activeStyle   = azdo.ActiveStyle.Copy()
	inactiveStyle = azdo.InactiveStyle.Copy()
)

type model struct {
	prTextarea         textarea.Model
	repo               *git.Repository
	gitStatus          string
	azdo               *azdo.Model
	activeSection      activeSection
	prOrPipelineChoice list.Model
	sections           []sections.Section
}

type activeSection int

const (
	prOrPipelineSection activeSection = iota
	openPRSection
	azdoSection
)

func (m *model) setAzdoClientFromRemote(branch string) {
	remotes, err := m.repo.Remotes()
	if err != nil {
		panic(err)
	}
	remote := remotes[0].Config().URLs[0]

	u, err := url.Parse(remote)
	if err != nil {
		panic(err)
	}
	parts := strings.Split(u.Path, "/")
	organization := parts[1]
	project := parts[2]
	repository := parts[4]
	m.azdo = azdo.New(organization, project, repository, branch, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
}

func initialModel() model {
	commitSection := sections.NewCommitSection()
	commitSection.Show()
	commitSection.Focus()
	worktreeSection := sections.NewWorktreeSection()
	worktreeSection.Section.Show()
	_, err := worktreeSection.Repo.Head()
	if err != nil {
		panic(err)
	}
	log2file("initialModel")
	return model{
		sections: []sections.Section{commitSection, worktreeSection.Section},
	}
}

type InitializedMsg bool

func (m *model) Init() tea.Cmd {
	log2file("Init")
	m.prTextarea = textarea.New()
	m.prTextarea.Placeholder = "Title and description"
	m.prTextarea.SetPromptFunc(5, func(i int) string {
		if i == 0 {
			return "Title:"
		} else {
			return " Desc:"
		}
	})
	// branch := ref.Name()
	// m.setAzdoClientFromRemote(branch.String())
	m.prOrPipelineChoice = list.New([]list.Item{listitems.StagedFileItem{Name: "Open PR"}, listitems.StagedFileItem{Name: "Go to pipelines"}}, listitems.GitItemDelegate{}, 20, 20)
	return func() tea.Msg {
		return InitializedMsg(true)
	}
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
			return m, nil
		}
	case InitializedMsg:
		log2file("InitializedMsg")
		return m, nil
	case tea.WindowSizeMsg:
		sections.ActiveStyle.Height(msg.Height - 2)
		sections.InactiveStyle.Height(msg.Height - 2)
		for _, section := range m.sections {
			section.SetDimensions(msg.Width, msg.Height)
		}
		return m, nil
	}
	var sections []sections.Section
	for _, section := range m.sections {
		if !section.IsHidden() {
			sec, cmd := section.Update(msg)
			sections = append(sections, sec)
			cmds = append(cmds, cmd)
		}
	}
	m.sections = sections
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
	for _, section := range m.sections {
		log2file("Viewing section")
		if section.IsHidden() {
			continue
		}
		view = attachView(view, section.View())
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
	for i, section := range m.sections {
		if section.IsFocused() {
			section.Blur()
			if i+1 == len(m.sections) {
				m.sections[0].Focus()
				m.activeSection = activeSection(i)
				return
			}
			m.sections[i+1].Focus()
			m.activeSection = activeSection(i + 1)
			return
		}
	}
}
