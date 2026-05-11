package server

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const pullInterval = 5 * time.Minute

// startAutoSync pulls immediately then every 5 minutes while the server runs.
// compassPath is read fresh each tick via the getter so config changes are respected.
func startAutoSync(getCompassPath func() string) {
	go func() {
		gitPull(getCompassPath())
		for range time.NewTicker(pullInterval).C {
			gitPull(getCompassPath())
		}
	}()
}

// gitPull runs git pull on the compass folder if it is a git repository.
func gitPull(compassPath string) {
	if compassPath == "" || !isGitRepo(compassPath) {
		return
	}
	cmd := exec.Command("git", "-C", compassPath, "pull", "--rebase")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("git pull: %v\n%s", err, out)
		return
	}
	log.Printf("git pull: %s", compassPath)
}

// gitSync commits and pushes the compass folder if it is a git repository.
// It runs in the background so it never blocks an HTTP response.
func gitSync(compassPath, message string) {
	go func() {
		if !isGitRepo(compassPath) {
			return
		}

		cmds := [][]string{
			{"git", "-C", compassPath, "add", "-A"},
			{"git", "-C", compassPath, "commit", "--allow-empty-message", "-m", message},
			{"git", "-C", compassPath, "push"},
		}

		for _, args := range cmds {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Env = append(os.Environ(),
				"GIT_AUTHOR_NAME=compass-dash",
				"GIT_AUTHOR_EMAIL=compass-dash@local",
				"GIT_COMMITTER_NAME=compass-dash",
				"GIT_COMMITTER_EMAIL=compass-dash@local",
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("git sync: %s: %v\n%s", args[0], err, out)
				if args[2] == "commit" {
					continue
				}
				return
			}
		}
		log.Printf("git sync: pushed compass at %s", time.Now().Format(time.RFC3339))
	}()
}

func isGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
