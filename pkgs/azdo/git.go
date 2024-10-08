package azdo

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type GitClientInterface interface {
	CreatePullRequest(context.Context, git.CreatePullRequestArgs) (git.GitPullRequest, error)
}

type GitClient struct {
	git.Client
}

func NewGitClient(ctx context.Context, orgurl, projectid, pat string) GitClientInterface {
	azdoconn := azuredevops.NewPatConnection(orgurl, pat)
	client, err := git.NewClient(ctx, azdoconn)
	if err != nil {
		panic(fmt.Sprintf("failed to create git client: %v", err))
	}
	return &GitClient{
		Client: client,
	}
}

func (g *GitClient) CreatePullRequest(ctx context.Context, args git.CreatePullRequestArgs) (git.GitPullRequest, error) {
	pr, err := g.Client.CreatePullRequest(ctx, args)
	if err != nil {
		return git.GitPullRequest{}, err
	}
	return *pr, nil
}
