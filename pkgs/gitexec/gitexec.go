// by the time of writing this, I've tried to use go-git and git2go to manage git repositories.
// Both of them did not work as expected.
// git2go needs CGO to be enabled apperentely and go-git had bugs on windows regarding line endings and .gitignore (most likely because of line endings too).
// So I've decided to use the git command line tool to manage git repositories.

package gitexec

import (
	"azdoext/pkgs/logger"
	"os/exec"
	"strings"
)

type GitFile struct {
	Name      string
	Staged    bool
	RawStatus string
	Change    string
}

type GitConfig struct {
	Origin        string
	CurrentBranch string
}

func Config() GitConfig {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	origin, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	cmd = exec.Command("git", "branch", "--show-current")
	currentBranch, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	return GitConfig{
		Origin:        strings.TrimSpace(string(origin)),
		CurrentBranch: strings.TrimSpace(string(currentBranch)),
	}
}

func Status() []GitFile {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	return parseStatus(string(out))
}

func parseStatus(status string) []GitFile {
	files := []GitFile{}
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, GitFile{
			Name:      line[3:],
			Staged:    line[0] != ' ' && line[0] != '?',
			RawStatus: line,
			Change:    line[0:2],
		})
	}
	return files
}

func AddGlob(glob string) {
	logger := logger.NewLogger("gitexec.log")
	logger.LogToFile("debug", "Adding files with glob: "+glob)
	cmd := exec.Command("git", "add", glob)
	out, err := cmd.CombinedOutput()
	logger.LogToFile("debug", string(out))
	if err != nil {
		panic(err)
	}
}

func Add(file string) {
	cmd := exec.Command("git", "add", file)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Unstage(file string) {
	cmd := exec.Command("git", "restore", "--staged", file)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Commit(message string) {
	logger := logger.NewLogger("gitexec.log")
	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	logger.LogToFile("debug", string(out))
	if err != nil {
		panic(err)
	}

}

func Push(remote string, branch string, credential string) {
	// Construct the remote URL with the PAT
	remoteWithPAT := strings.Replace(remote, "https://", "https://"+credential+"@", 1)

	cmd := exec.Command("git", "push", remoteWithPAT, branch)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Pull(remote string, branch string) {
	cmd := exec.Command("git", "pull", remote, branch)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

}
