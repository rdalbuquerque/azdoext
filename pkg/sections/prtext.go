package sections

import (
	"azdoext/pkg/azdo"
	"azdoext/pkg/logger"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"
	"azdoext/pkg/utils"
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type PRSection struct {
	errorDisplayed    bool
	logger            *logger.Logger
	hidden            bool
	focused           bool
	title             string
	textarea          textarea.Model
	project           string
	repositoryId      uuid.UUID
	currentBranch     string
	defaultBranch     string
	gitclient         azdo.GitClientInterface
	sectionIdentifier SectionName
	help              string
}

func (pr *PRSection) IsHidden() bool {
	return pr.hidden
}

func (pr *PRSection) IsFocused() bool {
	return pr.focused
}

func NewPRSection(secid SectionName, gitclient azdo.GitClientInterface, azdoconfig azdo.Config) Section {
	logger := logger.NewLogger("pr.log")
	title := "Open PR:"
	styledHelpText := styles.ShortHelpStyle.Render("ctrl+s save and open PR")
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
		logger:            logger,
		title:             title,
		textarea:          textarea,
		sectionIdentifier: secid,
		project:           azdoconfig.ProjectId,
		repositoryId:      azdoconfig.RepositoryId,
		currentBranch:     formatBranchName(azdoconfig.CurrentBranch),
		defaultBranch:     azdoconfig.DefaultBranch,
		gitclient:         gitclient,
		help:              styledHelpText,
	}
}

// formatBranchName adds refs/heads/ prefix to the branch name if it is not already present
func formatBranchName(branch string) string {
	if strings.HasPrefix(branch, "refs/heads/") {
		return branch
	}
	return "refs/heads/" + branch
}

func (pr *PRSection) GetSectionIdentifier() SectionName {
	return pr.sectionIdentifier
}

func (pr *PRSection) SetDimensions(width, height int) {
	pr.textarea.SetWidth(styles.DefaultSectionWidth + 20)
	pr.textarea.SetHeight(height - 4)
}

func (pr *PRSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if pr.focused {
			switch msg.String() {
			case "ctrl+s":
				if pr.textarea.Focused() {
					pr.textarea.Blur()
				}
				return pr, func() tea.Msg { return teamsg.SubmitPRMsg(pr.textarea.Value()) }
			}
			if pr.errorDisplayed && msg.String() == "enter" {
				pr.textarea.Reset()
				return pr, nil
			}

		}
	case teamsg.SubmitPRMsg:
		titleAndDescription := strings.SplitN(string(msg), "\n", 2)
		title := titleAndDescription[0]
		if title == "" {
			panic("PR title cannot be empty")
		}
		var description string
		if len(titleAndDescription) == 2 {
			description = titleAndDescription[1]
		}
		pr.logger.LogToFile("info", fmt.Sprintf("submitting PR with title: %s and description: %s, from %s to %s", title, description, pr.currentBranch, pr.defaultBranch))
		return pr, func() tea.Msg { return pr.openPR(pr.currentBranch, pr.defaultBranch, title, description) }
	case teamsg.PRErrorMsg:
		pr.textarea.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Red))
		pr.textarea.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Red))
		pr.textarea.Focus()
		pr.errorDisplayed = true
		pr.textarea.SetValue(string(msg) + "\nPress 'enter' to dismiss")
	}
	ta, cmd := pr.textarea.Update(msg)
	pr.textarea = ta
	return pr, cmd
}

func (pr *PRSection) openPR(currentBranch, defaultBranch, title, description string) tea.Msg {
	pr.logger.LogToFile("info", fmt.Sprintf("creating PR with title: %s and description: %s, from %s to %s", title, description, currentBranch, defaultBranch))
	createdpr, err := pr.gitclient.CreatePullRequest(context.Background(), git.CreatePullRequestArgs{
		RepositoryId: utils.Ptr(pr.repositoryId.String()),
		Project:      &pr.project,
		GitPullRequestToCreate: &git.GitPullRequest{
			Title:         &title,
			Description:   &description,
			SourceRefName: &currentBranch,
			TargetRefName: &defaultBranch,
		},
	})
	pr.logger.LogToFile("info", fmt.Sprintf("PR created: %v", createdpr))
	if err != nil {
		pr.logger.LogToFile("error", "error while creating PR: "+err.Error())
		return teamsg.PRErrorMsg(err.Error())
	}
	return teamsg.GitPRCreatedMsg{}
}

func (pr *PRSection) View() string {
	title := styles.TitleStyle.Render(pr.title)
	if !pr.hidden {
		if pr.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, title, "", pr.textarea.View(), "", pr.help))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, title, "", pr.textarea.View(), "", pr.help))
	}
	return ""
}

func (pr *PRSection) Hide() {
	pr.focused = false
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
