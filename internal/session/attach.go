package session

import (
	"fmt"
	"os"
	"os/exec"
)

func Attach(sessionName string) error {
	if err := exec.Command("tmux", "has-session", "-t", sessionName).Run(); err != nil {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	// If inside tmux, switch-client; otherwise attach-session.
	if os.Getenv("TMUX") != "" {
		cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
