package save

import (
	"fmt"
	"time"

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

	ts := time.Now().Format("15:04:05")
	content := fmt.Sprintf("## Assistant (%s)\n\n%s\n\n", ts, msg)

	return appendToFile(notePath, content)
}
