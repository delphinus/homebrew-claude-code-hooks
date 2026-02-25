package save

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/filter"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

func handlePostToolUse(input *hookdata.HookInput) error {
	switch input.ToolName {
	case "Bash":
		return handleBash(input)
	case "EnterPlanMode":
		return handleEnterPlanMode(input)
	case "ExitPlanMode":
		return handleExitPlanMode(input)
	case "Edit", "Write":
		return handleEditWrite(input)
	default:
		return nil
	}
}

func handleBash(input *hookdata.HookInput) error {
	command := input.ToolInput.Command
	if command == "" {
		return nil
	}

	if !filter.ShouldRecordCommand(command) {
		return nil
	}

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(notePath); err != nil {
		return nil
	}

	ts := time.Now().Format("15:04:05")

	title := fmt.Sprintf("Command (%s)", ts)
	if input.ToolInput.Description != "" {
		title = fmt.Sprintf("%s (%s)", input.ToolInput.Description, ts)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "> [!terminal]- %s\n", title)
	b.WriteString("> ```bash\n")
	for _, line := range strings.Split(command, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	b.WriteString("> ```\n\n")

	return appendToFile(notePath, b.String())
}

func handleEnterPlanMode(input *hookdata.HookInput) error {
	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(notePath); err != nil {
		return nil
	}

	vaultDir := config.VaultDir()

	// Try to add related links if not already present
	content, err := os.ReadFile(notePath)
	if err == nil && !strings.Contains(string(content), "\nrelated:") {
		_ = note.AddPreviousLinks(notePath, input.SessionID, vaultDir)
	}

	// Create plan flag
	cacheDir := config.CacheDir()
	planFlagPath := filepath.Join(cacheDir, input.SessionID+"-in-plan")
	_ = os.WriteFile(planFlagPath, []byte{}, 0o644)

	ts := time.Now().Format("15:04:05")
	return appendToFile(notePath, fmt.Sprintf("> [!plan] Entering Plan Mode (%s)\n\n", ts))
}

func handleExitPlanMode(input *hookdata.HookInput) error {
	cacheDir := config.CacheDir()
	planCachePath := filepath.Join(cacheDir, input.SessionID+"-plan")
	planFlagPath := filepath.Join(cacheDir, input.SessionID+"-in-plan")

	var plan string
	if data, err := os.ReadFile(planCachePath); err == nil {
		plan = string(data)
		os.Remove(planCachePath)
	}
	os.Remove(planFlagPath)

	if plan == "" {
		return nil
	}

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(notePath); err != nil {
		return nil
	}

	ts := time.Now().Format("15:04:05")

	// Use first line as plan title
	planTitle := "Plan"
	lines := strings.SplitN(plan, "\n", 2)
	if len(lines) > 0 {
		title := strings.TrimLeft(lines[0], "# ")
		if title != "" {
			planTitle = title
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "> [!plan]- %s (%s)\n", planTitle, ts)
	for _, line := range strings.Split(plan, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	b.WriteString("\n")

	return appendToFile(notePath, b.String())
}

func handleEditWrite(input *hookdata.HookInput) error {
	filePath := input.ToolInput.FilePath
	if filePath == "" {
		return nil
	}

	// If in plan mode and this is a Write, cache the content for ExitPlanMode
	if input.ToolName == "Write" {
		cacheDir := config.CacheDir()
		planFlagPath := filepath.Join(cacheDir, input.SessionID+"-in-plan")
		if _, err := os.Stat(planFlagPath); err == nil {
			planCachePath := filepath.Join(cacheDir, input.SessionID+"-plan")
			_ = os.WriteFile(planCachePath, []byte(input.ToolInput.Content), 0o644)
		}
	}

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(notePath); err != nil {
		return nil
	}

	ts := time.Now().Format("15:04:05")

	// Convert to relative path if under CWD
	displayPath := filePath
	if input.CWD != "" && strings.HasPrefix(filePath, input.CWD+"/") {
		displayPath = strings.TrimPrefix(filePath, input.CWD+"/")
	}

	return appendToFile(notePath, fmt.Sprintf("> [!file] %s: %s (%s)\n\n", input.ToolName, displayPath, ts))
}

func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
