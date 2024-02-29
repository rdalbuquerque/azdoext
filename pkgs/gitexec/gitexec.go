// by the time of writing this, I've tried to use go-git and git2go to manage git repositories.
// Both of them did not work as expected.
// git2go needs CGO to be enabled apperentely and go-git had bugs on windows regarding line endings and .gitignore (most likely because of line endings too).
// So I've decided to use the git command line tool to manage git repositories.

package gitexec

import (
	"fmt"
	"os"
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
	log2file(fmt.Sprintf("origin: %s", string(origin)))
	cmd = exec.Command("git", "branch", "--show-current")
	currentBranch, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	log2file(fmt.Sprintf("currentBranch: %s", string(currentBranch)))
	return GitConfig{
		Origin:        strings.TrimSpace(string(origin)),
		CurrentBranch: strings.TrimSpace(string(currentBranch)),
	}
}

func Status() []GitFile {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.CombinedOutput()
	log2file(string(out))
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
	cmd := setupCommand("git", "add", glob)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Add(file string) {
	cmd := setupCommand("git", "add", file)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Unstage(file string) {
	cmd := setupCommand("git", "restore", "--staged", file)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Commit(message string) {
	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	log2file(string(out))
	if err != nil {
		log2file(fmt.Sprintf("commit msg: %s", message))
		log2file(fmt.Sprintf("commit error: %s", err))
		panic(err)
	}

}

func Push(remote string, branch string) {
	cmd := setupCommand("git", "push", remote, branch)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func Pull(remote string, branch string) {
	cmd := setupCommand("git", "pull", remote, branch)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

}

func setupCommand(args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)

	stdoutFile, err := os.OpenFile("stdout.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer stdoutFile.Close()
	cmd.Stdout = stdoutFile

	stderrFile, err := os.OpenFile("stderr.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer stderrFile.Close()
	cmd.Stderr = stderrFile

	return cmd
}

func log2file(msg string) {
	f, err := os.OpenFile("gitexec.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(msg + "\n"); err != nil {
		fmt.Println(err)
	}
}
