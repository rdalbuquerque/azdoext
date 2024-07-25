package azdo

import (
	"azdoext/pkgs/utils"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type Config struct {
	OrgUrl         string
	OrgName        string
	AccoundId      string
	ProjectName    string
	ProjectId      string
	PAT            string
	RepositoryName string
	RepositoryId   uuid.UUID
	CurrentBranch  string
	DefaultBranch  string
}

func GetAzdoConfig(remoteUrl string, currentBranch string) Config {
	orgurl := getOrgUrl(remoteUrl)
	conn := azuredevops.NewPatConnection(orgurl, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
	azdopat := os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")
	projectname := getProjectName(remoteUrl)
	reponame := getRepositoryName(remoteUrl)
	return Config{
		OrgUrl:         orgurl,
		ProjectName:    projectname,
		RepositoryName: reponame,
		ProjectId:      getProjectId(conn, projectname),
		RepositoryId:   getRepositoryId(conn, projectname, reponame),
		PAT:            azdopat,
	}
}

func getOrgUrl(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	orgurl := strings.Join(remoteurl_parts[:4], "/")
	return orgurl
}

func getProjectName(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	projectname := remoteurl_parts[4]
	return projectname
}

func getRepositoryName(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	reponame := remoteurl_parts[6]
	return reponame
}

func getProjectId(conn *azuredevops.Connection, projectname string) string {
	client, err := core.NewClient(context.Background(), conn)
	if err != nil {
		panic(fmt.Errorf("failed to get core client: %v", err))
	}
	project, err := client.GetProject(context.Background(), core.GetProjectArgs{
		ProjectId: utils.Ptr(projectname),
	})
	if err != nil {
		panic(fmt.Errorf("failed to get project: %v", err))
	}
	projectid := project.Id.String()
	return projectid
}

func getRepositoryId(conn *azuredevops.Connection, projectname string, reponame string) uuid.UUID {
	client, err := git.NewClient(context.Background(), conn)
	if err != nil {
		panic(fmt.Errorf("failed to get git client: %v", err))
	}
	repo, err := client.GetRepository(context.Background(), git.GetRepositoryArgs{
		Project:      utils.Ptr(projectname),
		RepositoryId: utils.Ptr(reponame),
	})
	if err != nil {
		panic(fmt.Errorf("failed to get repository: %v", err))
	}
	return *repo.Id
}
