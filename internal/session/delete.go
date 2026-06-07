package session

import (
	"fmt"
	"os/exec"
	"strings"
)

func Delete(sessionName string) error {
	if err := exec.Command("tmux", "has-session", "-t", sessionName).Run(); err != nil {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	// Parse agent/<repo>/<branch> to extract branch and repo path.
	parts := strings.SplitN(sessionName, "/", 3)
	if len(parts) < 3 {
		return fmt.Errorf("unexpected session name format: %s", sessionName)
	}
	branch := parts[2]

	// Get the session's working directory to derive the repo path.
	out, err := exec.Command("tmux", "display-message", "-t", sessionName, "-p", "#{session_path}").Output()
	if err != nil {
		return fmt.Errorf("could not determine session path: %w", err)
	}
	worktreePath := strings.TrimSpace(string(out))

	// The parent repo is one level up from the worktree (wt convention).
	repoPath := parentRepo(worktreePath)

	// Kill the tmux session first.
	if out, err := exec.Command("tmux", "kill-session", "-t", sessionName).CombinedOutput(); err != nil {
		return fmt.Errorf("tmux kill-session: %s", strings.TrimSpace(string(out)))
	}

	// Remove the worktree via wt.
	wtCmd := exec.Command("wt", "remove", branch)
	wtCmd.Dir = repoPath
	if out, err := wtCmd.CombinedOutput(); err != nil {
		fmt.Printf("warning: failed to remove worktree %s: %s. Clean up manually with: wt remove %s\n",
			branch, strings.TrimSpace(string(out)), branch)
	}

	return nil
}

func DeleteWithConfirmation(sessionName string, confirm func(string) bool) error {
	if !confirm(fmt.Sprintf("Delete session %s? This will remove the worktree. [y/N]", sessionName)) {
		return nil
	}
	return Delete(sessionName)
}

// parentRepo walks up from the worktree path to find the parent repo.
// wt places worktrees as siblings of the main repo by default, so we
// look for the nearest ancestor that contains a .git directory.
func parentRepo(worktreePath string) string {
	out, err := exec.Command("git", "-C", worktreePath, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return worktreePath
	}
	// The first worktree entry is always the main repo.
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			return strings.TrimPrefix(line, "worktree ")
		}
	}
	return worktreePath
}
