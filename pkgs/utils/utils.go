package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func Ptr[T any](v T) *T {
	return &v
}

type AzdoInfo struct {
	OrgUrl         string
	Project        string
	RepositoryName string
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

func ExtractAzdoInfo(remoteUrl string) AzdoInfo {
	remoteUrl = strings.Trim(remoteUrl, "/")
	orgUrl := strings.Join(strings.Split(remoteUrl, "/")[:4], "/")
	project := strings.Split(remoteUrl, "/")[4]
	repositoryName := strings.Split(remoteUrl, "/")[len(strings.Split(remoteUrl, "/"))-1]
	return AzdoInfo{
		OrgUrl:         orgUrl,
		Project:        project,
		RepositoryName: repositoryName,
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
		return string(*result)
	}
	return string(*status)
}
