package teamsg

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/utils"

	"github.com/charmbracelet/bubbles/list"
	"github.com/google/uuid"
)

type AzdoConfigMsg azdo.Config
type viewportContentMsg string
type OptionsMsg []list.Item
type SubmitChoiceMsg listitems.OptionName
type CommitMsg string
type ToggleMaximizeMsg struct{}
type ReadLogsCtxDoneMsg struct{}
type BuildsFetchedMsg []list.Item
type PipelineSelectedMsg listitems.PipelineItem
type PipelineRunIdMsg struct {
	RunId        int
	PipelineName string
	ProjectId    string
	Status       string
}

type RecordSelectedMsg struct {
	RecordId uuid.UUID
}

type PipelineRunStateMsg struct {
	Items  []list.Item
	Status string
}
type GitPRCreatedMsg bool
type SubmitPRMsg string
type PRErrorMsg string
type GitPushedMsg bool
type GitPushingMsg bool
type NothingToCommitMsg struct{}
type LogMsg struct {
	utils.TimelineRecordId
	StepRecordId uuid.UUID
	BuildStatus  string
	BuildResult  string
	NewContent   string
}

type BuildStatusUpdateMsg struct {
	Status string
	Result string
}
