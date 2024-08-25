package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func Ptr[T any](v T) *T {
	return &v
}

func SleepWithContext(ctx context.Context, wait time.Duration) error {
	sleepDone := make(chan struct{})
	go func() {
		time.Sleep(wait)
		close(sleepDone)
	}()

	select {
	case <-sleepDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context was done during wait period: %w", ctx.Err())
	}
}

func GetRepositoryId(ctx context.Context, azdoconn *azuredevops.Connection, project, repositoryName string) uuid.UUID {
	gitclient, err := git.NewClient(ctx, azdoconn)
	if err != nil {
		panic(err)
	}
	repos, err := gitclient.GetRepositories(ctx, git.GetRepositoriesArgs{Project: &project})
	if err != nil {
		panic(err)
	}
	for _, repo := range *repos {
		if *repo.Name == repositoryName {
			return *repo.Id
		}
	}
	return uuid.Nil
}

func GetRepositoryDefaultBranch(ctx context.Context, azdoconn *azuredevops.Connection, project, repositoryName string) string {
	gitclient, err := git.NewClient(ctx, azdoconn)
	if err != nil {
		panic(err)
	}
	repos, err := gitclient.GetRepositories(ctx, git.GetRepositoriesArgs{Project: &project})
	if err != nil {
		panic(err)
	}
	for _, repo := range *repos {
		if *repo.Name == repositoryName {
			return *repo.DefaultBranch
		}
	}
	return ""
}

// ref: https://forum.golangbridge.org/t/generic-and-typecasting/29903/5
type Status interface {
	~string
}

type Result interface {
	~string
}

func StatusOrResult[S Status, R Result](status *S, result *R) string {
	if result != nil {
		if *result != "" {
			return string(*result)
		}
	}
	return string(*status)
}

type TimelineRecordId string

type StepRecordId string

type Logs map[StepRecordId]string

type LogMsg struct {
	TimelineRecordId
	StepRecordId
	BuildStatus  string
	BuildResult  string
	NewContent   string
	ReadLogError error
}
