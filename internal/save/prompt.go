package save

import (
	"fmt"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

func handleUserPromptSubmit(input *hookdata.HookInput) error {
	prompt := input.Prompt

	// Skip internal notifications (background task results etc.)
	if strings.Contains(prompt, "<task-notification>") {
		return nil
	}

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, prompt)
	if err != nil {
		return err
	}

	// Re-post plan from previous session if available
	repostPlan(notePath)

	prompt = EnsureTableBlankLines(prompt)

	ts := time.Now().Format("15:04:05")
	content := fmt.Sprintf("## User (%s)\n\n%s\n\n", ts, prompt)

	return appendToFile(notePath, content)
}
