package tabcolor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// userVarName is the WezTerm user var that holds the current Claude Code state.
// The wezterm.lua side scans each tab's panes for this user var to color the tab.
const userVarName = "claude_state"

// validStates is the whitelist of states that map to tab colors on the WezTerm side.
// "default" clears the coloring (back to the normal tab color).
var validStates = map[string]bool{
	"startup":  true,
	"thinking": true,
	"idle":     true,
	"waiting":  true,
	"default":  true,
}

// Run sets the WezTerm user var claude_state for the current pane, so the tab can
// be colored according to the Claude Code state.
//
// WezTerm has no CLI to set a user var; the only mechanism is the OSC 1337
// SetUserVar escape sequence written to the pane's terminal. A hook's stdout is
// captured by Claude Code (and /dev/tty is unavailable), so when stdout is not a
// terminal we resolve the pane's tty device via `wezterm cli list` (keyed by
// WEZTERM_PANE) and write the sequence there. The user var then syncs across the
// mux to the GUI client, where format-tab-title reads it.
//
// It is a no-op outside WezTerm. Since this is purely cosmetic, all failures are
// swallowed so the hook never disrupts the Claude Code flow.
func Run(state string) error {
	if !validStates[state] {
		return fmt.Errorf("unknown state: %q", state)
	}

	pane := os.Getenv("WEZTERM_PANE")
	if pane == "" {
		return nil
	}

	seq := fmt.Sprintf(
		"\x1b]1337;SetUserVar=%s=%s\a",
		userVarName,
		base64.StdEncoding.EncodeToString([]byte(state)),
	)

	// Interactive invocation: stdout is the terminal, write directly.
	if fi, err := os.Stdout.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		_, _ = os.Stdout.WriteString(seq)
		return nil
	}

	// Hook invocation: stdout is captured. Resolve the pane's tty and write there.
	tty := paneTTY(pane)
	if tty == "" {
		return nil
	}
	f, err := os.OpenFile(tty, os.O_WRONLY, 0)
	if err != nil {
		return nil
	}
	defer f.Close()
	_, _ = f.WriteString(seq)
	return nil
}

type paneInfo struct {
	PaneID  int    `json:"pane_id"`
	TTYName string `json:"tty_name"`
}

// paneTTY returns the tty device path for the given WezTerm pane id,
// or "" if it cannot be determined.
func paneTTY(pane string) string {
	out, err := exec.Command("wezterm", "cli", "list", "--format", "json").Output()
	if err != nil {
		return ""
	}
	var panes []paneInfo
	if err := json.Unmarshal(out, &panes); err != nil {
		return ""
	}
	for _, p := range panes {
		if fmt.Sprintf("%d", p.PaneID) == pane {
			return p.TTYName
		}
	}
	return ""
}
