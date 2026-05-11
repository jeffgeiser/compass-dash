package server

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

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
				// commit exits 1 when there's nothing to commit — that's fine
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
