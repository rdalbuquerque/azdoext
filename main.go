package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"net/url"

	"explore-bubbletea/pkgs/azdo"

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

type model struct {
	textarea  textarea.Model
	worktree  *git.Worktree
	repo      *git.Repository
	gitStatus string
	spinner   spinner.Model // Add this line
	pushing   bool          // Add this line
	pushed    bool
	azdo      *azdo.Model
}

func (m *model) setAzdoClientFromRemote() {
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
	m.azdo = azdo.New(organization, project, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
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
		m.setAzdoClientFromRemote()
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
		case tea.KeyEnter:
			if m.pushed {
				i, ok := m.azdo.PipelineList.SelectedItem().(azdo.PipelineItem)
				if ok {
					m.gitStatus = "Pipeline selected: " + string(i.Title)
					return m, tea.Batch(func() tea.Msg { return m.azdo.RunOrFollowPipeline(i.Desc.(int), false) }, m.spinner.Tick)
				} else {
					m.gitStatus = "No pipeline selected"
					return m, nil
				}
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
			return m, m.azdo.FetchPipelines
		}
		m.gitStatus = string(msg)
	case azdo.PipelinesFetchedMsg:
		m.azdo.Update(msg)
		return m, nil
	}
	if m.pushing {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	textarea, txtcmd := m.textarea.Update(msg)
	m.textarea = textarea
	cmds = append(cmds, cmd, txtcmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.pushing {
		m.gitStatus = lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), "Pushing...\n")
	}
	if m.pushed {
		return lipgloss.JoinVertical(lipgloss.Top, m.gitStatus, m.azdo.PipelineList.View())
	}
	return lipgloss.JoinVertical(lipgloss.Top, "Git Commit", m.textarea.View(), m.gitStatus)
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
