package tabcolor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/kitty"
)

// userVarName is the WezTerm user var that holds the current Claude Code state.
// The wezterm.lua side scans each tab's panes for this user var to color the tab.
const userVarName = "claude_state"

// validStates is the whitelist of states that map to tab colors.
// "default" clears the coloring (back to the normal tab color).
var validStates = map[string]bool{
	"startup":  true,
	"thinking": true,
	"idle":     true,
	"waiting":  true,
	"default":  true,
}

// Run colors the current tab according to the Claude Code state.
//
// Two terminals are supported and probed in this order:
//
//   - kitty: `kitten @ set-tab-color` sets the tab colors directly, so nothing
//     needs to be written to the pane's terminal at all.
//   - WezTerm: there is no CLI to set a user var, so the OSC 1337 SetUserVar
//     escape sequence has to be written to the pane's terminal, and the
//     wezterm.lua side turns that into a color.
//
// It is a no-op under any other terminal. Since this is purely cosmetic, all
// failures are swallowed so the hook never disrupts the Claude Code flow.
func Run(state string) error {
	if !validStates[state] {
		return fmt.Errorf("unknown state: %q", state)
	}
	if window := os.Getenv("KITTY_WINDOW_ID"); window != "" {
		return runKitty(state, window)
	}
	if pane := os.Getenv("WEZTERM_PANE"); pane != "" {
		return runWezTerm(state, pane)
	}
	return nil
}

//: kitty

// tabColors holds the four colors kitty needs per state. They reproduce what
// wezterm.lua's tab_title.lua computed at render time:
//
//	active   = the state color as-is
//	inactive = the same color darkened by 25% (wezterm's Color:darken(0.25),
//	           which is lighten(-0.25), i.e. HSL lightness * 0.75)
//	fg       = #1a1b26 when the resulting lightness is > 0.4, else #c0caf5
//
// Every state stays above that threshold even after darkening, so the light
// foreground never comes up in practice; it is left out rather than carried
// over as dead configuration.
type tabColors struct{ activeBG, inactiveBG string }

const tabFG = "#1a1b26"

var stateColors = map[string]tabColors{
	"startup":  {"#7dcfff", "#1eacff"},
	"thinking": {"#bb9af7", "#7c3df0"},
	"idle":     {"#9ece6a", "#77b03a"},
	"waiting":  {"#e0af68", "#cc8a2a"},
}

func runKitty(state, window string) error {
	kitten := kitty.Path()
	if kitten == "" {
		return nil
	}
	// Match the tab *containing* this window. The hook runs as a descendant of
	// the Claude Code window, so KITTY_WINDOW_ID identifies it, and
	// KITTY_LISTEN_ON (also inherited) tells kitten which instance to talk to.
	args := []string{"@", "set-tab-color", "--match", "window_id:" + window}
	if c, ok := stateColors[state]; ok {
		args = append(args,
			"active_bg="+c.activeBG, "active_fg="+tabFG,
			"inactive_bg="+c.inactiveBG, "inactive_fg="+tabFG,
		)
	} else {
		// "default": revert to the colors from kitty.conf.
		args = append(args,
			"active_bg=NONE", "active_fg=NONE",
			"inactive_bg=NONE", "inactive_fg=NONE",
		)
	}
	_ = exec.Command(kitten, args...).Run()
	return nil
}

//: WezTerm

func runWezTerm(state, pane string) error {
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

	// Hook invocation: stdout is captured by Claude Code (and /dev/tty is
	// unavailable). Resolve the pane's tty and write there instead. The user var
	// then syncs across the mux to the GUI client, where format-tab-title reads it.
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
