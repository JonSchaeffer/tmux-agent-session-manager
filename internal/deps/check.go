package deps

import (
	"fmt"
	"os/exec"
	"strings"
)

var required = []string{"tmux", "fzf", "wt", "git"}

func Check() error {
	var missing []string
	for _, bin := range required {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		msg := "missing required dependencies: " + strings.Join(missing, ", ")
		if contains(missing, "wt") {
			msg += "\n  wt (worktrunk): brew install worktrunk && wt config shell install"
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func CheckTmuxRunning() error {
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		if strings.Contains(stderr, "no server running") ||
			strings.Contains(stderr, "error connecting") ||
			strings.Contains(stderr, "No such file") {
			return fmt.Errorf("tmux server is not running. Start tmux first.")
		}
		// list-sessions exits 1 when there are no sessions — that's fine.
	}
	return nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
