package save

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

func handleStop(input *hookdata.HookInput) error {
	msg := input.LastAssistantMessage

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, msg)
	if err != nil {
		return err
	}

	if msg == "" {
		return nil
	}

	// Check if already recorded by PostToolUse's recordLastAssistantMessage
	cacheDir := config.CacheDir()
	lastMsgCachePath := filepath.Join(cacheDir, input.SessionID+"-last-msg")
	if cached, err := os.ReadFile(lastMsgCachePath); err == nil {
		if string(cached) == msg {
			return nil
		}
	}

	msg = EnsureTableBlankLines(msg)

	ts := time.Now().Format("15:04:05")
	content := fmt.Sprintf("## Assistant (%s)\n\n%s\n\n", ts, msg)

	if err := appendToFile(notePath, content); err != nil {
		return err
	}

	// Update cache to prevent duplicate recording
	return os.WriteFile(lastMsgCachePath, []byte(msg), 0o644)
}
