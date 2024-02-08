package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"explore-bubbletea/pkgs/azdo"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type gitOutputMsg string
type gitErrorMsg string

type model struct {
	textarea        textarea.Model
	worktree        *git.Worktree
	repo            *git.Repository
	gitStatus       string
	spinner         spinner.Model // Add this line
	pushing         bool          // Add this line
	pushed          bool
	pipelines       list.Model
	pipelineRunning bool
	azdoClient      *azdo.AzdoClient
}

func getUrlFromRemote

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Your commit message here"
	ti.Focus()
	azdoClient := azdo.NewAzdoClient(os.Getenv("AZDO_ORG_URL"), os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
	return model{
		textarea:  ti,
		azdoClient: azdoClient,
		gitStatus: "preparing git",
	}
}

func (m *model) Init() tea.Cmd {
	// reset log file
	_ = os.Remove("log.txt")
	m.spinner = spinner.New()       // Initialize the spinner
	m.spinner.Spinner = spinner.Dot // Set the spinner style
	m.pipelines = list.New(nil, list.DefaultDelegate{}, 0, 0)
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
		case tea.KeyEnter:
			if m.pushed {
				i, ok := m.pipelines.SelectedItem().(item)
				if ok {
					m.gitStatus = "Pipeline selected: " + string(i.name)
					m.pipelineRunning = true
					return m, tea.Batch(m.runPipeline(i.id), m.spinner.Tick)
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
			return m, m.fetchPipelines
		}
		m.gitStatus = string(msg)
	}
	if m.pushing || m.pipelineRunning {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	textarea, txtcmd := m.textarea.Update(msg)
	m.textarea = textarea
	pipelines, listcmd := m.pipelines.Update(msg)
	m.pipelines = pipelines
	cmds = append(cmds, cmd, txtcmd, listcmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.pipelineRunning {
		m.gitStatus = lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), "Pipeline running...\n")
		return m.gitStatus
	}
	if m.pushing {
		m.gitStatus = lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), "Pushing...\n")
	}
	if m.pushed {
		return lipgloss.JoinVertical(lipgloss.Top, m.gitStatus, m.pipelines.View())
	}
	return lipgloss.JoinVertical(lipgloss.Top, titleStyle.Render("Git Commit"), m.textarea.View(), m.gitStatus)
}

func (m *model) push() tea.Msg {
	err := m.repo.Push(&git.PushOptions{
		Auth:     &githttp.BasicAuth{Username: "", Password: os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")},
		Progress: nil,
	})
	if err != nil {
		log(err.Error())
		return gitErrorMsg(err.Error())
	} else {
		log("Pushed")
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
func log(msg string) {
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(msg + "\n"); err != nil {
		fmt.Println(err)
	}
}
