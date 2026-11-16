package notify

import (
	"fmt"
	"os"
	"os/exec"
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
// Only kitty is supported. In every other case (another terminal, helper
// missing, or the helper fails) it falls back to a plain osascript
// notification, so a notification is always shown.
func Run(title, message string) error {
	if window := os.Getenv("KITTY_WINDOW_ID"); window != "" {
		label, focused := kittyTab(window)
		if focused {
			return nil
		}
		return post(title, label, message, window)
	}

	return osascriptNotify(title, "", message)
}

// post prefers the clickable native helper and falls back to osascript.
func post(title, subtitle, message, pane string) error {
	if path, err := exec.LookPath(helperBinary); err == nil {
		if err := runHelper(path, title, subtitle, message, pane); err == nil {
			return nil
		}
		// fall through to osascript on failure
	}
	return osascriptNotify(title, subtitle, message)
}

// helperArgs builds the argument list for the native notifier helper.
// An empty subtitle is omitted entirely (older helpers ignore unknown flags).
// The socket (KITTY_LISTEN_ON) is passed via the inherited environment, not here.
func helperArgs(title, subtitle, message, pane string) []string {
	args := []string{"post", "--title", title}
	if subtitle != "" {
		args = append(args, "--subtitle", subtitle)
	}
	return append(args, "--message", message, "--pane", pane)
}

func runHelper(path, title, subtitle, message, pane string) error {
	cmd := exec.Command(path, helperArgs(title, subtitle, message, pane)...)
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
