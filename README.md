
# azdoext

A terminal UI, powered by [bubbletea framework](https://github.com/charmbracelet/bubbleteahttps://github.com/charmbracelet/bubbletea)\
to help streamline the process of commiting, pushing, creating PRs and following pipelines in Azure DevOps.

## Prerequisites

- environment variable `AZDO_PERSONAL_ACCESS_TOKEN` with full access to **all accessible organizations**
	- this is necessarry because the app fetches the account id during initialization. [ref](https://medium.com/@shivapatel1102001/get-list-of-organization-from-azure-devops-microsoft-account-861ea29dae93)
- git installed - there is a [go package to handle git operations](https://pkg.go.dev/github.com/go-git/go-git/v5) but it still has a few bugs, so it just run git commands.

## Get started
### Install on Linux and macOS
```bash
curl -fsSL https://raw.githubusercontent.com/rdalbuquerque/azdoext/main/scripts/install.sh | sh
```

### Install on Windows
```powershell
Invoke-RestMethod "https://raw.githubusercontent.com/rdalbuquerque/azdoext/main/scripts/install.ps1" | Invoke-Expression
```

### Keybindings
- `ctrl+c`: quit
- `ctrl+b`: go back to previous page
- `ctrl+h`: show/hide help
- `ctrl+r`: restart the process
- `ctrl+s`: save on any textarea
	- on commit message: push
		- if no files are staged, stage all files before pushing
	- on Pull Request section: open a new PR and go to the pipelines
- `ctrl+a`: stage/unstage file on status list
- `tab`: switch between available sections
- `enter`: select an option on any list (has no effect on file status list)
- `/` : search for a string while on pipeline logs
- `f` : toggle follow on a pipeline run that is in progress

## Pages and sections
The app is divided into pages and sections:
* git page: where you can stage files, commit, push and create PRs. There are sections such as commit, git status and PR
* pipeline list: where you can see all pipelines related to the current repository and go to the tasks of the last run or execute a new run
* pipeline run: where you can see and follow the logs and tasks of a specific pipeline run. This page contains a section for the pipeline tasks and one for the logs of each task
* help: full instructions

## Commit, push and open a PR
When the app starts you will see two sections, commit message and the changed files, files staged will be shown in green.\
Here you can either write a commit msg and hit `ctrl+s` to save, stage all files and push\
You can also stage individual files by pressing `ctrl+a` while the file is selected.

After changes are pushed you will presented with a choice, you can either go directly to pipelines or open a PR.\
If you chose to open a PR, you will be presented with a text area where the first line is PR title and the rest is PR description.\
To save and open the PR press `ctrl+s`.\
With a opened PR you are taken to pipelines section.
OBS: Currently, only PRs to the default branch are supported.

## List pipelines and execute new runs
On pipelines page, you will see all pipelines related to you current repository and their last run status.\
When you press enter you will be presented with a choice, go to the tasks of the selected pipeline or execute a new run.\
While on the pipeline instance section you can go to logs, browse and hit `/` to search for a specific string.\
If the selected run is in progress, you'll see the live pipeline logs, you can hit `f` to toggle follow.\
If enabled, you will see the latest logs and the task list cursor will indicate the current running task.

## Demo

https://github.com/rdalbuquerque/azdoext/assets/23347635/d5e7357c-fe93-4d36-90e1-746354c17a16

