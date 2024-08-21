package azdo

import (
	"azdoext/pkgs/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	orgname := getOrgName(remoteUrl)
	conn := azuredevops.NewPatConnection(orgurl, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
	azdopat := os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")
	projectname := getProjectName(remoteUrl)
	reponame := getRepositoryName(remoteUrl)
	return Config{
		AccoundId:      getAccountId(orgname, azdopat),
		OrgUrl:         orgurl,
		OrgName:        orgname,
		ProjectName:    projectname,
		RepositoryName: reponame,
		ProjectId:      getProjectId(conn, projectname),
		RepositoryId:   getRepositoryId(conn, projectname, reponame),
		PAT:            azdopat,
		CurrentBranch:  currentBranch,
		DefaultBranch:  getDefaultBranch(conn, projectname, reponame),
	}
}

func getDefaultBranch(conn *azuredevops.Connection, projectname, reponame string) string {
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
	if repo.DefaultBranch == nil {
		panic("default branch not found")
	}
	return *repo.DefaultBranch
}

func getUserId(authHeader string) string {
	url := "https://app.vssps.visualstudio.com/_apis/profile/profiles/me?api-version=7.1"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", authHeader)

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var currentUser struct {
		Id string `json:"id"`
	}
	json.Unmarshal(body, &currentUser)
	return currentUser.Id
}

type accounts struct {
	Count int       `json:"count"`
	Value []account `json:"value"`
}

type account struct {
	AccountId   string `json:"accountId"`
	AccountUri  string `json:"accountUri"`
	AccountName string `json:"accountName"`
}

func getAccountId(orgName, pat string) string {
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(":"+pat)))
	userid := getUserId(authHeader)
	if userid == "" {
		panic("user id not found")
	}
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/accounts?memberId=%s&api-version=7.1", userid)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", authHeader)

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var accounts accounts
	json.Unmarshal(body, &accounts)
	for _, account := range accounts.Value {
		if strings.EqualFold(account.AccountName, orgName) {
			return account.AccountId
		}
	}
	panic("account not found")
}

func getOrgName(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	orgname := remoteurl_parts[3]
	return orgname
}

func getOrgUrl(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	orgurl := strings.Join(remoteurl_parts[:4], "/")
	return orgurl
}

func getProjectName(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	projectname, err := url.QueryUnescape(remoteurl_parts[4])
	if err != nil {
		panic(fmt.Errorf("failed to unescape project name: %v", err))
	}
	return projectname
}

func getRepositoryName(remoteUrl string) string {
	remoteurl_parts := strings.Split(remoteUrl, "/")
	reponame, err := url.QueryUnescape(remoteurl_parts[6])
	if err != nil {
		panic(fmt.Errorf("failed to unescape repository name: %v", err))
	}
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
