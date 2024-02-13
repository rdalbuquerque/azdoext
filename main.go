package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"net/url"

	"explore-bubbletea/pkgs/azdo"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type gitOutputMsg string
type gitErrorMsg string

var (
	activeStyle   = azdo.ActiveStyle.Copy()
	InactiveStyle = azdo.InactiveStyle.Copy()
)

type model struct {
	textarea       textarea.Model
	worktree       *git.Worktree
	repo           *git.Repository
	gitStatus      string
	spinner        spinner.Model // Add this line
	pushing        bool          // Add this line
	pushed         bool
	azdo           *azdo.Model
	activeSection  activeSection
	stagedFileList list.Model
}

type activeSection int

const (
	gitcommitSection activeSection = iota
	worktreeSection
	prOrPipelineSection
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
	ti := textarea.New()
	ti.Placeholder = "Your commit message here"
	ti.Focus()
	return model{
		textarea:  ti,
		gitStatus: "preparing git",
	}
}

func (m *model) Init() tea.Cmd {
	// reset log file
	_ = os.Remove("log.txt")
	m.spinner = spinner.New()       // Initialize the spinner
	m.spinner.Spinner = spinner.Dot // Set the spinner style
	r, err := git.PlainOpen(".")
	if err != nil {
		panic(err)
	}
	w, err := r.Worktree()
	if err != nil {
		panic(err)
	}
	m.stagedFileList = setStagedFileList(w)
	gitStatus, err := w.Status()
	if err != nil {
		panic(err)
	}
	m.worktree = w
	m.repo = r

	ref, err := r.Head()
	if err != nil {
		panic(err)
	}
	branch := ref.Name()
	m.setAzdoClientFromRemote(branch.String())

	return func() tea.Msg {
		return gitOutputMsg(gitStatus.String())
	}
}

// Main update function.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.azdo.SetHeights(msg.Height - 2)
		activeStyle.Height(msg.Height - 2)
		InactiveStyle.Height(msg.Height - 2)
		m.textarea.SetHeight(msg.Height - 4)
		m.stagedFileList.SetHeight(msg.Height - 2)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyEnter:
			log2file("Enter key pressed")
			if m.textarea.Focused() {
				textarea, txtcmd := m.textarea.Update(msg)
				m.textarea = textarea
				cmds = append(cmds, txtcmd)
			}
			if m.pushed {
				azdo, azdocmd := m.azdo.Update(msg)
				cmds = append(cmds, azdocmd)
				m.azdo = azdo
			}
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			if m.activeSection == gitcommitSection {
				m.activeSection = worktreeSection
				m.textarea.Blur()
			} else {
				m.textarea.Focus()
				m.activeSection = gitcommitSection
			}
		case tea.KeyCtrlA:
			if m.activeSection == worktreeSection {
				m.addToStage()
			}
			fileList, cmd := m.stagedFileList.Update(msg)
			m.stagedFileList = fileList
			return m, cmd
		case tea.KeyCtrlS:
			m.textarea.Blur()
			if m.worktree != nil {
				_, err := m.worktree.Commit(m.textarea.Value(), &git.CommitOptions{
					Author: &object.Signature{
						Name:  "name",
						Email: "email",
						When:  time.Now(),
					},
				})
				if err != nil {
					m.gitStatus = err.Error()
					return m, nil
				}

				m.gitStatus = "Changes committed"

				m.pushing = true
				return m, tea.Batch(m.push, m.spinner.Tick)
			} else {
				m.gitStatus = "Worktree is not initialized"
				return m, nil
			}
		}

	case gitErrorMsg:
		m.gitStatus = string(msg)
		m.textarea.Blur()
	case gitOutputMsg:
		if msg == "Pushed" {
			m.pushed = true
			m.pushing = false
			return m, m.azdo.FetchPipelines(0)
		}
		m.gitStatus = string(msg)
	case azdo.PipelinesFetchedMsg, azdo.PipelineIdMsg, azdo.PipelineStateMsg:
		azdo, cmd := m.azdo.Update(msg)
		m.azdo = azdo
		return m, cmd
	}
	if m.pushing {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.azdo != nil && m.pushed {
		azdo, cmd := m.azdo.Update(msg)
		m.azdo = azdo
		return m, cmd
	}
	if m.activeSection == worktreeSection {
		m.stagedFileList, cmd = m.stagedFileList.Update(msg)
		return m, cmd
	}
	textarea, txtcmd := m.textarea.Update(msg)
	m.textarea = textarea
	cmds = append(cmds, txtcmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.pushed {
		return m.azdo.View()
	}
	var gitCommitView, worktreeView string
	gitCommitSection := lipgloss.JoinVertical(lipgloss.Top, "Git commit:", m.textarea.View())
	if m.activeSection == gitcommitSection {
		gitCommitView = activeStyle.Render(gitCommitSection)
		worktreeView = InactiveStyle.Render(m.stagedFileList.View())
	} else if m.activeSection == worktreeSection {
		gitCommitView = InactiveStyle.Render(gitCommitSection)
		worktreeView = activeStyle.Render(m.stagedFileList.View())
	}
	if m.pushing {
		m.stagedFileList.Title = "Pushing"
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, gitCommitView, "  ", worktreeView)
}

func (m *model) push() tea.Msg {
	err := m.repo.Push(&git.PushOptions{
		Auth:     &githttp.BasicAuth{Username: "", Password: os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")},
		Progress: nil,
	})
	if err != nil {
		log2file(err.Error())
		return gitErrorMsg(err.Error())
	} else {
		log2file("Pushed")
		return gitOutputMsg("Pushed")
	}
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
