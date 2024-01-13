package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type gitOutputMsg string
type gitErrorMsg string

type model struct {
	textarea  textarea.Model
	worktree  *git.Worktree
	repo      *git.Repository
	gitStatus string
	spinner   spinner.Model // Add this line
	pushing   bool          // Add this line
	pushed    bool
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
	m.spinner = spinner.New()       // Initialize the spinner
	m.spinner.Spinner = spinner.Dot // Set the spinner style
	return func() tea.Msg {
		r, err := git.PlainOpen(".")
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		w, err := r.Worktree()
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		err = w.AddGlob(".")
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		gitStatus, err := w.Status()
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		m.worktree = w
		m.repo = r
		return gitOutputMsg(gitStatus.String())
	}
}

// Main update function.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
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
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	case gitErrorMsg:
		m.gitStatus = string(msg)
		m.textarea.Blur()
	case gitOutputMsg:
		if msg == "Pushed" {
			m.pushed = true
			m.pushing = false
		}
		m.gitStatus = string(msg)
	}
	if m.pushing {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.pushing {
		m.gitStatus = lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), "Pushing...\n")
	}
	if m.pushed {
		return lipgloss.JoinVertical(lipgloss.Top, m.gitStatus)
	}
	return lipgloss.JoinVertical(lipgloss.Top, m.gitStatus, m.textarea.View())
}

func (m *model) push() tea.Msg {
	err := m.repo.Push(&git.PushOptions{
		Auth:     &http.BasicAuth{Username: "", Password: os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")},
		Progress: nil,
	})
	if err != nil {
		return gitErrorMsg(err.Error())
	} else {
		return gitOutputMsg("Pushed")
	}
}

func main() {
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
