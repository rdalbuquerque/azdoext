package sections

import (
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type GitPRCreatedMsg bool

type PRSection struct {
	logger        *logger.Logger
	hidden        bool
	focused       bool
	title         string
	textarea      textarea.Model
	project       string
	repositoryId  uuid.UUID
	currentBranch string
	defaultBranch string
	gitclient     git.Client
}

func (pr *PRSection) IsHidden() bool {
	return pr.hidden
}

func (pr *PRSection) IsFocused() bool {
	return pr.focused
}

func NewPRSection(_ context.Context) Section {
	logger := logger.NewLogger("pr.log")
	title := "Open PR:"
	textarea := textarea.New()
	textarea.SetHeight(styles.ActiveStyle.GetHeight() - 2)
	textarea.Placeholder = "Title and description"
	textarea.SetPromptFunc(6, func(i int) string {
		if i == 0 {
			return "Title:"
		} else {
			return " Desc:"
		}
	})
	return &PRSection{
		logger:   logger,
		title:    title,
		textarea: textarea,
	}
}

func (pr *PRSection) SetDimensions(width, height int) {
	pr.textarea.SetWidth(styles.DefaultSectionWidth)
	pr.textarea.SetHeight(height - 1)
}

func (pr *PRSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case GitInfoMsg:
		azdoInfo := utils.ExtractAzdoInfo(msg.RemoteUrl)
		azdoconn := azuredevops.NewPatConnection(azdoInfo.OrgUrl, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
		gitclient, err := git.NewClient(context.Background(), azdoconn)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return pr, nil
			}
			panic(err)
		}
		defaultBranch := utils.GetRepositoryDefaultBranch(context.Background(), azdoconn, azdoInfo.Project, azdoInfo.RepositoryName)
		pr.logger.LogToFile("info", "default branch: "+defaultBranch)
		repositoryId := utils.GetRepositoryId(context.Background(), azdoconn, azdoInfo.Project, azdoInfo.RepositoryName)
		pr.logger.LogToFile("info", "repository id: "+repositoryId.String())
		pr.gitclient, pr.project, pr.currentBranch, pr.defaultBranch, pr.repositoryId = gitclient, azdoInfo.Project, msg.CurrentBranch, defaultBranch, repositoryId
		return pr, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if pr.textarea.Focused() {
				pr.textarea.Blur()
			}
			return pr, func() tea.Msg { return SubmitPRMsg(pr.textarea.Value()) }
		}
	case SubmitPRMsg:
		titleAndDescription := strings.SplitN(string(msg), "\n", 2)
		if len(titleAndDescription) != 2 {
			return pr, func() tea.Msg { return PRErrorMsg("Title and description are required") }
		}
		title := titleAndDescription[0]
		description := titleAndDescription[1]
		pr.logger.LogToFile("info", fmt.Sprintf("submitting PR with title: %s and description: %s, from %s to %s", title, description, pr.currentBranch, pr.defaultBranch))
		return pr, func() tea.Msg { return pr.openPR(pr.currentBranch, pr.defaultBranch, title, description) }
	case PRErrorMsg:
		pr.textarea.Placeholder = string(msg)
	}
	ta, cmd := pr.textarea.Update(msg)
	pr.textarea = ta
	return pr, cmd
}

func (pr *PRSection) openPR(currentBranch, defaultBranch, title, description string) tea.Msg {
	_, err := pr.gitclient.CreatePullRequest(context.Background(), git.CreatePullRequestArgs{
		RepositoryId: utils.Ptr(pr.repositoryId.String()),
		Project:      &pr.project,
		GitPullRequestToCreate: &git.GitPullRequest{
			Title:         &title,
			Description:   &description,
			SourceRefName: &currentBranch,
			TargetRefName: &defaultBranch,
		},
	})
	if err != nil {
		pr.logger.LogToFile("error", "error while creating PR: "+err.Error())
		return GitPRCreatedMsg(false)
	}
	return GitPRCreatedMsg(true)
}

func (pr *PRSection) View() string {
	if !pr.hidden {
		if pr.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pr.title, pr.textarea.View()))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pr.title, pr.textarea.View()))
	}
	return ""
}

func (pr *PRSection) Hide() {
	pr.hidden = true
}

func (pr *PRSection) Show() {
	pr.hidden = false
}

func (pr *PRSection) Focus() {
	pr.Show()
	pr.textarea.Focus()
	pr.focused = true
}

func (pr *PRSection) Blur() {
	pr.textarea.Blur()
	pr.focused = false
}

type SubmitPRMsg string
type PRErrorMsg string
