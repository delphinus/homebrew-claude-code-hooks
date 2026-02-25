package save

import (
	"fmt"
	"os"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
)

// Run handles the save subcommand, dispatching to the appropriate handler
// based on the hook event name.
func Run(input *hookdata.HookInput) error {
	// Prevent recursion from claude -p calls during SessionEnd
	if os.Getenv("CLAUDE_OBSIDIAN_SAVING") != "" {
		return nil
	}

	if input.SessionID == "" {
		return nil
	}

	vaultDir := config.VaultDir()
	if err := os.MkdirAll(vaultDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Vault ディレクトリが見つかりません: %s\n", vaultDir)
		return nil
	}

	if err := os.MkdirAll(config.CacheDir(), 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	switch input.HookEventName {
	case "PostToolUse":
		return handlePostToolUse(input)
	case "UserPromptSubmit":
		return handleUserPromptSubmit(input)
	case "Stop":
		return handleStop(input)
	case "SessionEnd":
		return handleSessionEnd(input)
	default:
		return nil
	}
}
