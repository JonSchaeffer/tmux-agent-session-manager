package session

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type CreateOpts struct {
	RepoPath string
	RepoName string
	Branch   string
	Agent    string
}

var validBranch = regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`)

func Create(opts CreateOpts) error {
	if err := validateRepo(opts.RepoPath); err != nil {
		return err
	}
	if opts.Branch == "" {
		return fmt.Errorf("branch name is required")
	}
	if !validBranch.MatchString(opts.Branch) {
		return fmt.Errorf("invalid branch name %q: only letters, digits, '.', '-', '_', '/' are allowed", opts.Branch)
	}

	// Fetch — warn on failure, don't abort.
	fetchCmd := exec.Command("git", "-C", opts.RepoPath, "fetch")
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: git fetch failed: %s\n", strings.TrimSpace(string(out)))
	}

	// Create worktree via wt.
	wtCmd := exec.Command("wt", "switch", "--create", opts.Branch)
	wtCmd.Dir = opts.RepoPath
	var wtStderr bytes.Buffer
	wtCmd.Stderr = &wtStderr
	if err := wtCmd.Run(); err != nil {
		if isNotFound(err) {
			return fmt.Errorf("wt (worktrunk) is required. Install: brew install worktrunk && wt config shell install")
		}
		return fmt.Errorf("wt switch --create %s: %s", opts.Branch, strings.TrimSpace(wtStderr.String()))
	}

	// Find the new worktree path.
	worktreePath, err := findWorktreePath(opts.RepoPath, opts.Branch)
	if err != nil {
		return fmt.Errorf("could not determine worktree path: %w", err)
	}

	// Create the tmux session.
	sessionName := fmt.Sprintf("ai/%s/%s", opts.RepoName, opts.Branch)
	checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if checkCmd.Run() == nil {
		return fmt.Errorf("session %s already exists", sessionName)
	}

	newCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", worktreePath)
	if out, err := newCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux new-session: %s", strings.TrimSpace(string(out)))
	}

	// Store the agent name as a session environment variable for reliable retrieval.
	exec.Command("tmux", "set-environment", "-t", sessionName, "TAISM_AGENT", opts.Agent).Run()

	// Launch agent inside the session.
	sendCmd := exec.Command("tmux", "send-keys", "-t", sessionName, opts.Agent, "Enter")
	if out, err := sendCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux send-keys: %s", strings.TrimSpace(string(out)))
	}

	return nil
}

func validateRepo(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("repo path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo path is not a directory: %s", path)
	}
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("not a git repository: %s", path)
	}
	return nil
}

func findWorktreePath(repoPath, branch string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", err
	}

	// Porcelain format blocks separated by blank lines:
	//   worktree /path/to/worktree
	//   HEAD <sha>
	//   branch refs/heads/<name>
	var currentPath string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if line == "branch refs/heads/"+branch {
			return currentPath, nil
		}
	}
	return "", fmt.Errorf("worktree for branch %q not found", branch)
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "executable file not found")
}
