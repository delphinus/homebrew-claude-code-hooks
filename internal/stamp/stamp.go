// Package stamp records which Claude Code session lives in which kitty window.
//
// kitty forgets its layout when it exits, and on a machine that gets shut down
// every evening that happens nightly. The conversations themselves survive in
// the transcripts and can be brought back with `claude --resume <id>`, but
// nothing knows which tab was holding which conversation.
//
// A session id is only ever visible in a hook's JSON, so SessionStart writes it
// onto the window as a user var. The side that dumps the layout (session_state.py
// in the dotfiles) picks up only the windows carrying this var.
package stamp

import (
	"os"
	"os/exec"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/kitty"
)

// UserVarName is the kitty user var holding the session id of the Claude Code
// process running in that window.
const UserVarName = "claude_session"

// Run stamps the current kitty window with the given session id.
//
// It is a no-op outside kitty, and every failure is swallowed. A missing stamp
// costs one tab in tomorrow's restored layout, which is never worth interrupting
// the session over.
func Run(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return set(UserVarName + "=" + sessionID)
}

// Clear removes the stamp from the current kitty window.
//
// SessionEnd needs this because the window outlives the conversation: Claude Code
// is started from a login shell that stays behind when it exits, so a window whose
// conversation was closed on purpose would otherwise keep its session id and be
// resurrected in tomorrow's layout. A window that dies with the conversation still
// running keeps its stamp, which is exactly the case worth restoring.
func Clear() error {
	// kitten unsets a variable when given just its name.
	return set(UserVarName)
}

func set(arg string) error {
	// The window the hook belongs to. kitten would resolve the current window
	// from this same variable, but matching on it explicitly keeps the target
	// unambiguous even when the hook is spawned from a deeper subprocess.
	window := os.Getenv("KITTY_WINDOW_ID")
	if window == "" {
		return nil
	}
	kitten := kitty.Path()
	if kitten == "" {
		return nil
	}
	_ = exec.Command(kitten, "@", "set-user-vars", "--match", "id:"+window, arg).Run()
	return nil
}
