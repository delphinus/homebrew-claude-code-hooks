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
	content := userCallout(ts, prompt)

	return appendToFile(notePath, content)
}

// userCallout wraps a user prompt in an Obsidian callout so it stands out
// visually from assistant messages. Every content line is prefixed with "> "
// (blank lines become ">") to keep the callout continuous.
func userCallout(ts, prompt string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "> [!question] User (%s)\n", ts)
	for _, line := range strings.Split(prompt, "\n") {
		if line == "" {
			b.WriteString(">\n")
		} else {
			b.WriteString("> " + line + "\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}
