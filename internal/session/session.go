package session

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Session struct {
	Name      string    `json:"name"`
	RepoName  string    `json:"repo_name"`
	Branch    string    `json:"branch"`
	Agent     string    `json:"agent"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	WorkDir   string    `json:"work_dir"`
}

var shellProcesses = map[string]bool{
	"bash": true, "zsh": true, "sh": true, "fish": true,
}

var knownAgents = map[string]bool{
	"claude": true,
	"aider":  true,
	"codex":  true,
	"pi":     true,
}

func List() ([]Session, error) {
	out, err := exec.Command("tmux", "list-sessions",
		"-F", "#{session_name}\t#{session_created}\t#{session_path}").Output()
	if err != nil {
		if isNoServerErr(err) {
			return nil, fmt.Errorf("tmux is not running")
		}
		// No sessions returns exit 1 with a specific message — treat as empty.
		if strings.Contains(string(out), "no server running") ||
			strings.Contains(err.Error(), "exit status 1") {
			return nil, fmt.Errorf("tmux is not running")
		}
		return nil, fmt.Errorf("tmux list-sessions: %w", err)
	}

	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		name, createdStr, workDir := parts[0], parts[1], parts[2]

		if !strings.HasPrefix(name, "ai/") {
			continue
		}

		segments := strings.SplitN(name, "/", 3)
		if len(segments) < 3 {
			continue
		}
		repoName := segments[1]
		branch := segments[2]

		unix, err := strconv.ParseInt(createdStr, 10, 64)
		if err != nil {
			continue
		}
		createdAt := time.Unix(unix, 0)

		agent := detectAgent(name)
		status := detectStatus(name)

		sessions = append(sessions, Session{
			Name:      name,
			RepoName:  repoName,
			Branch:    branch,
			Agent:     agent,
			Status:    status,
			CreatedAt: createdAt,
			WorkDir:   workDir,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

func detectAgent(sessionName string) string {
	// Prefer the stored environment variable (set at creation time).
	out, err := exec.Command("tmux", "show-environment", "-t", sessionName, "TAISM_AGENT").Output()
	if err == nil {
		line := strings.TrimSpace(string(out))
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}

	// Fall back to process inspection.
	out, err = exec.Command("tmux", "list-panes", "-t", sessionName,
		"-F", "#{pane_current_command}").Output()
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		cmd := strings.ToLower(strings.TrimSpace(line))
		if knownAgents[cmd] {
			return cmd
		}
	}
	return "unknown"
}

func detectStatus(sessionName string) string {
	out, err := exec.Command("tmux", "list-panes", "-t", sessionName,
		"-F", "#{pane_current_command}").Output()
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		cmd := strings.ToLower(strings.TrimSpace(line))
		if shellProcesses[cmd] {
			return "idle"
		}
	}
	return "running"
}

func isNoServerErr(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	return strings.Contains(string(exitErr.Stderr), "no server running") ||
		strings.Contains(string(exitErr.Stderr), "error connecting")
}

func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func MarshalJSON(sessions []Session) ([]byte, error) {
	return json.MarshalIndent(sessions, "", "  ")
}
