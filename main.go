package main

import (
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
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type gitOutputMsg string
type gitErrorMsg string

type item struct {
	name string
	id   float64
}

func (i item) FilterValue() string { return i.name }
func (i item) Title() string       { return i.name }
func (i item) Description() string { return fmt.Sprintf("%f", i.id) }

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
}

func (m *model) runPipeline(pipelineId float64) tea.Cmd {
	return func() tea.Msg {
		remotes, err := m.repo.Remotes()
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		remote := remotes[0].Config().URLs[0]

		// Parse the remote URL
		u, err := url.Parse(remote)
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		log(u.Path)
		parts := strings.Split(u.Path, "/")
		organization := parts[1]
		project := parts[2]
		// Construct the Azure DevOps API URL
		apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines/%d/runs?api-version=7.1-preview.1", organization, project, int(pipelineId))
		log(apiURL)
		client := &http.Client{}
		req, err := http.NewRequest("POST", apiURL, nil)
		if err != nil {
			log(err.Error())
			return tea.Quit
		}
		// transform the PAT into a base64 string
		b64authstring := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "", os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))))
		log(b64authstring)
		req.Header.Set("Authorization", "Basic "+b64authstring)
		resp, err := client.Do(req)
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		log(resp.Status)
		if err != nil {
			return gitErrorMsg(err.Error())
		}
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		resultJson, _ := json.MarshalIndent(result, "", "  ")
		log(string(resultJson))
		m.pipelineRunning = true
		monitorPipeline(organization, project, pipelineId, result["id"].(float64), b64authstring)
		m.pipelineRunning = false
		return gitOutputMsg("pipeline completed")
	}
}

func monitorPipeline(organization string, project string, pipelineId float64, runId float64, b64authstring string) {
	client := &http.Client{}
	apiUrl := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines/%d/runs/%d?api-version=6.0-preview.1", organization, project, int(pipelineId), int(runId))
	log("Monitoring pipeline: " + apiUrl)
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log(err.Error())
	}
	req.Header.Set("Authorization", "Basic "+b64authstring)
	for {
		resp, err := client.Do(req)
		if err != nil {
			log(err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		log(resp.Status)
		if err != nil {
			log(err.Error())
		}
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		// check if pipeline is still running, if it's not, break
		status := result["status"].(string)
		if status == "completed" {
			break
		}
		log(status)
		time.Sleep(1 * time.Second)
	}
}

func (m *model) fetchPipelines() tea.Msg {
	remotes, err := m.repo.Remotes()
	if err != nil {
		return gitErrorMsg(err.Error())
	}
	remote := remotes[0].Config().URLs[0]

	// Parse the remote URL
	u, err := url.Parse(remote)
	if err != nil {
		return gitErrorMsg(err.Error())
	}
	log(u.Path)
	parts := strings.Split(u.Path, "/")
	organization := parts[1]
	project := parts[2]

	// Construct the Azure DevOps API URL
	apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines?api-version=6.0-preview.1", organization, project)
	log(apiURL)
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return gitErrorMsg(err.Error())
	}
	// transform the PAT into a base64 string
	b64authstring := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "", os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))))
	log(b64authstring)
	req.Header.Set("Authorization", "Basic "+b64authstring)
	resp, err := client.Do(req)
	if err != nil {
		return gitErrorMsg(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	log(resp.Status)
	log(string(body))
	if err != nil {
		return gitErrorMsg(err.Error())
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	items := []list.Item{}
	for _, pipeline := range result["value"].([]interface{}) {
		pipelineName := pipeline.(map[string]interface{})["name"].(string)
		pipelineId := pipeline.(map[string]interface{})["id"].(float64)
		items = append(items, item{name: pipelineName, id: pipelineId})
	}
	m.pipelines = list.New(items, itemDelegate{}, 0, 10)
	m.pipelines.Title = "Pipelines"
	return gitOutputMsg("Pipelines fetched")
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
	if m.pushing {
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
