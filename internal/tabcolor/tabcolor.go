package tabcolor

import (
	"fmt"
	"os"
	"os/exec"
)

// userVarName is the WezTerm user var that holds the current Claude Code state.
// The wezterm.lua side reads tab.active_pane.user_vars[userVarName] to color the tab.
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

// Run sets the WezTerm user var for the current pane to the given state,
// so the tab can be colored according to the Claude Code state.
//
// It is a no-op outside WezTerm (WEZTERM_PANE unset). Since this is purely
// cosmetic, failures of the underlying `wezterm cli` invocation are swallowed
// so the hook never disrupts the Claude Code flow.
func Run(state string) error {
	if !validStates[state] {
		return fmt.Errorf("unknown state: %q", state)
	}

	// Only meaningful inside WezTerm.
	if os.Getenv("WEZTERM_PANE") == "" {
		return nil
	}

	// Best-effort: ignore errors (e.g. wezterm not on PATH, mux unreachable).
	cmd := exec.Command("wezterm", "cli", "set-user-var", "--name", userVarName, "--value", state)
	_ = cmd.Run()
	return nil
}
