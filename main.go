package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"net/url"

	"explore-bubbletea/pkgs/azdo"
	"explore-bubbletea/pkgs/listitems"

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
	inactiveStyle = azdo.InactiveStyle.Copy()
)

type model struct {
	commitTextarea     textarea.Model
	prTextarea         textarea.Model
	worktree           *git.Worktree
	repo               *git.Repository
	gitStatus          string
	spinner            spinner.Model // Add this line
	pushing            bool          // Add this line
	pushed             bool
	azdo               *azdo.Model
	activeSection      activeSection
	stagedFileList     list.Model
	prOrPipelineChoice list.Model
}

type activeSection int

const (
	gitcommitSection activeSection = iota
	worktreeSection
	prOrPipelineSection
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
	ti := textarea.New()
	ti.Placeholder = "Your commit message here"
	ti.Focus()
	return model{
		commitTextarea: ti,
		gitStatus:      "preparing git",
	}
}

func (m *model) Init() tea.Cmd {
	// reset log file
	_ = os.Remove("log.txt")
	m.spinner = spinner.New()       // Initialize the spinner
	m.spinner.Spinner = spinner.Dot // Set the spinner style
	m.prTextarea = textarea.New()
	m.prTextarea.Placeholder = "Title and description"
	m.prTextarea.SetPromptFunc(5, func(i int) string {
		if i == 0 {
			return "Title:"
		} else {
			return " Desc:"
		}
	})

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
	m.prOrPipelineChoice = list.New([]list.Item{listitems.StagedFileItem{Name: "Open PR"}, listitems.StagedFileItem{Name: "Go to pipelines"}}, listitems.GitItemDelegate{}, 20, 20)
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
		inactiveStyle.Height(msg.Height - 2)
		m.commitTextarea.SetHeight(msg.Height - 4)
		m.prTextarea.SetHeight(msg.Height - 4)
		m.prOrPipelineChoice.SetHeight(msg.Height - 2)
		m.stagedFileList.SetHeight(msg.Height - 2)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.commitTextarea.Focused() {
				m.commitTextarea.Blur()
			}
		case tea.KeyEnter:
			log2file("Enter key pressed")
			if m.activeSection == prOrPipelineSection {
				if m.prOrPipelineChoice.SelectedItem().(listitems.StagedFileItem).Name == "Open PR" {
					m.activeSection = openPRSection
					m.prTextarea.Focus()
					return m, nil
				} else {
					m.activeSection = azdoSection
					return m, m.azdo.FetchPipelines(0)
				}
			}
			if m.commitTextarea.Focused() {
				textarea, txtcmd := m.commitTextarea.Update(msg)
				m.commitTextarea = textarea
				cmds = append(cmds, txtcmd)
			}
			if m.activeSection == openPRSection {
				prtext, prcmd := m.prTextarea.Update(msg)
				m.prTextarea = prtext
				return m, prcmd
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
				m.commitTextarea.Blur()
			} else {
				m.commitTextarea.Focus()
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
			if m.activeSection == openPRSection {
				titleAndDescription := strings.SplitN(m.prTextarea.Value(), "\n", 2)
				title := titleAndDescription[0]
				description := titleAndDescription[1]
				m.prTextarea.Blur()
				return m, tea.Batch(func() tea.Msg {
					return m.azdo.OpenPR(strings.Split(m.azdo.Branch, "/")[2], "master", title, description)
				}, m.azdo.FetchPipelines(0))
			}
			if m.commitTextarea.Focused() {
				m.commitTextarea.Blur()
				if m.worktree != nil {
					repo, err := m.repo.Config()
					if err != nil {
						panic(err)
					}
					authorName := repo.Author.Name
					authorEmail := repo.Author.Email
					if m.noStagedFiles() {
						m.addAllToStage()
						list, cmd := m.stagedFileList.Update(msg)
						m.stagedFileList = list
						cmds = append(cmds, cmd)
					}
					_, err = m.worktree.Commit(m.commitTextarea.Value(), &git.CommitOptions{
						Author: &object.Signature{
							Name:  authorName,
							Email: authorEmail,
							When:  time.Now(),
						},
					})
					if err != nil {
						m.gitStatus = err.Error()
						return m, nil
					}
					m.gitStatus = "Changes committed"
					m.pushing = true
					cmds = append(cmds, m.push, m.spinner.Tick)
					return m, tea.Batch(cmds...)
				} else {
					m.gitStatus = "Worktree is not initialized"
					return m, nil
				}
			}
		}

	case gitErrorMsg:
		m.gitStatus = string(msg)
		m.commitTextarea.Blur()
	case gitOutputMsg:
		if msg == "Pushed" {
			m.pushed = true
			m.pushing = false
			m.activeSection = prOrPipelineSection
			return m, nil
		}
		m.gitStatus = string(msg)
	case azdo.PipelinesFetchedMsg, azdo.PipelineIdMsg, azdo.PipelineStateMsg, azdo.PRMsg:
		if m.activeSection == openPRSection {
			m.activeSection = azdoSection
		}
		azdo, cmd := m.azdo.Update(msg)
		m.azdo = azdo
		return m, cmd
	}
	if m.pushing {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.activeSection == worktreeSection {
		m.stagedFileList, cmd = m.stagedFileList.Update(msg)
		return m, cmd
	}
	if m.activeSection == prOrPipelineSection {
		prOrPipelineChoice, cmd := m.prOrPipelineChoice.Update(msg)
		m.prOrPipelineChoice = prOrPipelineChoice
		return m, cmd
	}
	if m.activeSection == openPRSection {
		textarea, txtcmd := m.prTextarea.Update(msg)
		m.prTextarea = textarea
		return m, txtcmd
	}
	if m.activeSection == azdoSection {
		azdo, cmd := m.azdo.Update(msg)
		m.azdo = azdo
		return m, cmd
	}
	textarea, txtcmd := m.commitTextarea.Update(msg)
	m.commitTextarea = textarea
	cmds = append(cmds, txtcmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.activeSection == azdoSection {
		return m.azdo.View()
	}
	var gitCommitView, worktreeView string
	gitCommitSection := lipgloss.JoinVertical(lipgloss.Center, "Git commit:", m.commitTextarea.View())
	if m.activeSection == prOrPipelineSection {
		gitCommitView = inactiveStyle.Render(gitCommitSection)
		worktreeView = inactiveStyle.Render(m.stagedFileList.View())
		return lipgloss.JoinHorizontal(lipgloss.Left, gitCommitView, " ", worktreeView, " ", activeStyle.Render(m.prOrPipelineChoice.View()))
	}
	if m.activeSection == openPRSection {
		gitCommitView = inactiveStyle.Render(gitCommitSection)
		worktreeView = inactiveStyle.Render(m.stagedFileList.View())
		prCommitSection := lipgloss.JoinVertical(lipgloss.Center, "Open PR:", m.prTextarea.View())
		return lipgloss.JoinHorizontal(lipgloss.Left, gitCommitView, " ", worktreeView, " ", inactiveStyle.Render(m.prOrPipelineChoice.View()), " ", activeStyle.Render(prCommitSection))
	}
	if m.activeSection == gitcommitSection {
		gitCommitView = activeStyle.Render(gitCommitSection)
		worktreeView = inactiveStyle.Render(m.stagedFileList.View())
	} else if m.activeSection == worktreeSection {
		gitCommitView = inactiveStyle.Render(gitCommitSection)
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
