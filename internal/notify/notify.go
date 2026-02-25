package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes the notify subcommand.
// It shows a macOS notification, suppressing it if the current WezTerm pane is focused.
func Run(title, message string) error {
	// Check if running in WezTerm
	weztermPane := os.Getenv("WEZTERM_PANE")
	if weztermPane != "" {
		if shouldSuppress(weztermPane) {
			return nil
		}
	}

	// Display notification via osascript
	script := fmt.Sprintf(
		`display notification %q with title %q sound name "default"`,
		message, title,
	)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func shouldSuppress(weztermPane string) bool {
	// Get the frontmost process PID
	activePID := getFrontmostPID()
	if activePID == "" {
		return false
	}

	// Get the focused pane for that PID from WezTerm
	activePane := getWeztermFocusedPane(activePID)
	if activePane == "" {
		return false
	}

	return weztermPane == activePane
}

func getFrontmostPID() string {
	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to get the unix id of first process whose frontmost is true`)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type weztermClient struct {
	PID           int `json:"pid"`
	FocusedPaneID int `json:"focused_pane_id"`
}

func getWeztermFocusedPane(pid string) string {
	cmd := exec.Command("wezterm", "cli", "list-clients", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	var clients []weztermClient
	if err := json.Unmarshal(out, &clients); err != nil {
		return ""
	}

	for _, c := range clients {
		if fmt.Sprintf("%d", c.PID) == pid {
			return fmt.Sprintf("%d", c.FocusedPaneID)
		}
	}
	return ""
}
