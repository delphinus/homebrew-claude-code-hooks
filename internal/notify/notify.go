package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// helperBinary is the native (.app) helper installed alongside this binary. It
// posts a *clickable* macOS notification: clicking it (or opening the
// claude-code-hooks://activate?pane=N URL) focuses the originating pane.
const helperBinary = "claude-code-hooks-notify"

// Run executes the notify subcommand.
//
// It shows a macOS notification, suppressing it when the pane it came from is the
// one being looked at right now. The notification carries a subtitle with the
// originating tab ("<タブ番号>: <タブタイトル>") so it is obvious which session it
// came from, and when the native helper is installed it becomes clickable
// (click -> focus that pane).
//
// kitty and WezTerm are supported, probed in that order. In every other case
// (unknown terminal, helper missing, or the helper fails) it falls back to a plain
// osascript notification, so a notification is always shown.
func Run(title, message string) error {
	if window := os.Getenv("KITTY_WINDOW_ID"); window != "" {
		label, focused := kittyTab(window)
		if focused {
			return nil
		}
		return post(title, label, message, window, "kitty")
	}

	if pane := os.Getenv("WEZTERM_PANE"); pane != "" {
		if shouldSuppress(pane) {
			return nil
		}
		return post(title, tabLabel(pane), message, pane, "wezterm")
	}

	return osascriptNotify(title, "", message)
}

// post prefers the clickable native helper and falls back to osascript.
func post(title, subtitle, message, pane, term string) error {
	if path, err := exec.LookPath(helperBinary); err == nil {
		if err := runHelper(path, title, subtitle, message, pane, term); err == nil {
			return nil
		}
		// fall through to osascript on failure
	}
	return osascriptNotify(title, subtitle, message)
}

// helperArgs builds the argument list for the native notifier helper.
// An empty subtitle is omitted entirely (older helpers ignore unknown flags).
// The socket (KITTY_LISTEN_ON / WEZTERM_UNIX_SOCKET) is passed via the inherited
// environment, not here.
func helperArgs(title, subtitle, message, pane, term string) []string {
	args := []string{"post", "--title", title}
	if subtitle != "" {
		args = append(args, "--subtitle", subtitle)
	}
	return append(args, "--message", message, "--pane", pane, "--term", term)
}

func runHelper(path, title, subtitle, message, pane, term string) error {
	cmd := exec.Command(path, helperArgs(title, subtitle, message, pane, term)...)
	return cmd.Run()
}

func osascriptNotify(title, subtitle, message string) error {
	script := fmt.Sprintf(
		`display notification %q with title %q sound name "default"`,
		message, title,
	)
	if subtitle != "" {
		script = fmt.Sprintf(
			`display notification %q with title %q subtitle %q sound name "default"`,
			message, title, subtitle,
		)
	}
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
