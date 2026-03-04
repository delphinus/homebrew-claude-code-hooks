package save

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/filter"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

func handlePostToolUse(input *hookdata.HookInput) error {
	var err error
	switch input.ToolName {
	case "Bash":
		err = handleBash(input)
	case "EnterPlanMode":
		err = handleEnterPlanMode(input)
	case "ExitPlanMode":
		err = handleExitPlanMode(input)
	case "Edit", "Write":
		err = handleEditWrite(input)
	}
	if err != nil {
		return err
	}

	// Re-post plan from a previous session if available.
	// PostToolUse is the first event when Claude starts implementing a plan
	// in a new session (Edit, Write, Bash fire before any UserPromptSubmit).
	if cache, cacheErr := note.LoadSessionCache(input.SessionID); cacheErr == nil && cache != nil {
		repostPlan(cache.NotePath)
	}

	// Also record LastAssistantMessage if available (fallback for Stop event not firing)
	return recordLastAssistantMessage(input)
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
	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(notePath); err != nil {
		return nil
	}

	flushCachedPlan(input.SessionID, notePath)
	return nil
}

// flushCachedPlan writes any cached plan content to the note and cleans up
// cache files. Called from both handleExitPlanMode and handleSessionEnd to
// ensure plans are saved even when ExitPlanMode doesn't fire.
func flushCachedPlan(sessionID, notePath string) {
	cacheDir := config.CacheDir()
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	planFlagPath := filepath.Join(cacheDir, sessionID+"-in-plan")

	var plan string
	if data, err := os.ReadFile(planCachePath); err == nil {
		plan = string(data)
		os.Remove(planCachePath)
	}
	os.Remove(planFlagPath)

	if plan == "" {
		return
	}

	// Save a copy for re-posting in subsequent sessions, keyed by project
	// name so it works across different session IDs in the same project.
	// First line is the original note path, rest is the plan content.
	project := filepath.Base(filepath.Dir(notePath))
	planLastPath := filepath.Join(cacheDir, project+"-plan-last")
	_ = os.WriteFile(planLastPath, []byte(notePath+"\n"+plan), 0o644)

	ts := time.Now().Format("15:04:05")
	_ = appendToFile(notePath, formatPlanCallout(plan, ts, true))
}

// planRepostMaxAge is the maximum age of a plan-last cache file for it to be
// re-posted. If the file is older than this, it's considered stale and deleted
// without re-posting. This prevents plans from leaking into unrelated sessions
// that happen to be in the same project.
const planRepostMaxAge = 2 * time.Minute

// repostPlan re-posts the plan from a previous session into the current note.
// Uses a non-collapsible callout so the plan is immediately visible.
// Keyed by project name so it works across different session IDs.
// Only re-posts if the plan cache was created recently (within planRepostMaxAge).
func repostPlan(notePath string) {
	cacheDir := config.CacheDir()
	project := filepath.Base(filepath.Dir(notePath))
	planLastPath := filepath.Join(cacheDir, project+"-plan-last")

	info, err := os.Stat(planLastPath)
	if err != nil {
		return
	}

	// Discard stale plan cache
	if time.Since(info.ModTime()) > planRepostMaxAge {
		os.Remove(planLastPath)
		return
	}

	data, err := os.ReadFile(planLastPath)
	if err != nil {
		return
	}
	os.Remove(planLastPath)

	// First line is the original note path, rest is the plan content
	parts := strings.SplitN(string(data), "\n", 2)
	if len(parts) < 2 {
		return
	}
	origNotePath := parts[0]
	plan := parts[1]
	if plan == "" {
		return
	}

	// Add link to the original session's note
	if origNotePath != "" {
		if _, err := os.Stat(origNotePath); err == nil {
			origName := strings.TrimSuffix(filepath.Base(origNotePath), ".md")
			_ = appendToFile(notePath, fmt.Sprintf("> [!link] Plan from [[%s]]\n\n", origName))
		}
	}

	ts := time.Now().Format("15:04:05")
	_ = appendToFile(notePath, formatPlanCallout(plan, ts, false))
}

// formatPlanCallout formats a plan as an Obsidian callout.
// If collapsible is true, uses [!plan]- (collapsed by default).
// If false, uses [!plan] (always visible).
func formatPlanCallout(plan, ts string, collapsible bool) string {
	planTitle := "Plan"
	lines := strings.SplitN(plan, "\n", 2)
	if len(lines) > 0 {
		title := strings.TrimLeft(lines[0], "# ")
		if title != "" {
			planTitle = title
		}
	}

	var b strings.Builder
	if collapsible {
		fmt.Fprintf(&b, "> [!plan]- %s (%s)\n", planTitle, ts)
	} else {
		fmt.Fprintf(&b, "> [!plan] %s (%s)\n", planTitle, ts)
	}
	for _, line := range strings.Split(plan, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	b.WriteString("\n")
	return b.String()
}

// fileEntryPattern matches lines like:
//
//	> [!file] Edit: README.md (14:16:09)
//	> [!file] Edit: README.md (14:16:09) × 3
var fileEntryPattern = regexp.MustCompile(
	`^> \[!file\] (\w+): (.+) \(\d{2}:\d{2}:\d{2}\)(?: × (\d+))?$`,
)

// deduplicateFileEntry checks the last non-empty line of the note file. If it
// matches the same toolName and filePath, the line is replaced in-place with an
// incremented count (× N) and the function returns true. Otherwise it returns
// false so the caller can fall back to appending a new entry.
func deduplicateFileEntry(notePath, toolName, filePath string) (bool, error) {
	data, err := os.ReadFile(notePath)
	if err != nil {
		return false, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Find the last non-empty line.
	lastIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastIdx = i
			break
		}
	}
	if lastIdx < 0 {
		return false, nil
	}

	m := fileEntryPattern.FindStringSubmatch(lines[lastIdx])
	if m == nil {
		return false, nil
	}

	// m[1] = tool name, m[2] = file path, m[3] = optional count
	if m[1] != toolName || m[2] != filePath {
		return false, nil
	}

	count := 1
	if m[3] != "" {
		count, _ = strconv.Atoi(m[3])
	}
	count++

	// Rebuild the line preserving the original timestamp.
	// Extract everything up to and including the timestamp part.
	tsEnd := strings.LastIndex(lines[lastIdx], ")")
	if tsEnd < 0 {
		return false, nil
	}
	lines[lastIdx] = lines[lastIdx][:tsEnd+1] + fmt.Sprintf(" × %d", count)

	return true, os.WriteFile(notePath, []byte(strings.Join(lines, "\n")), 0o644)
}

func handleEditWrite(input *hookdata.HookInput) error {
	filePath := input.ToolInput.FilePath
	if filePath == "" {
		return nil
	}

	// If in plan mode, cache the content for ExitPlanMode/SessionEnd.
	// Write has content in tool_input; Edit needs to read from disk.
	cacheDir := config.CacheDir()
	planFlagPath := filepath.Join(cacheDir, input.SessionID+"-in-plan")
	if _, err := os.Stat(planFlagPath); err == nil {
		var content string
		if input.ToolName == "Write" {
			content = input.ToolInput.Content
		} else if input.ToolName == "Edit" {
			if data, err := os.ReadFile(filePath); err == nil {
				content = string(data)
			}
		}
		if content != "" {
			planCachePath := filepath.Join(cacheDir, input.SessionID+"-plan")
			_ = os.WriteFile(planCachePath, []byte(content), 0o644)
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

	if ok, err := deduplicateFileEntry(notePath, input.ToolName, displayPath); err == nil && ok {
		note.ActivateInObsidian(notePath)
		return nil
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
	if err != nil {
		return err
	}
	note.ActivateInObsidian(path)
	return nil
}

// recordLastAssistantMessage records the last assistant message if it hasn't been recorded yet.
// This serves as a fallback for when the Stop event doesn't fire.
func recordLastAssistantMessage(input *hookdata.HookInput) error {
	msg := input.LastAssistantMessage
	if msg == "" {
		return nil
	}

	// Check if we've already recorded this message
	cacheDir := config.CacheDir()
	lastMsgCachePath := filepath.Join(cacheDir, input.SessionID+"-last-msg")

	if cached, err := os.ReadFile(lastMsgCachePath); err == nil {
		if string(cached) == msg {
			// Already recorded this message
			return nil
		}
	}

	notePath, err := note.GetOrCreateNote(input.SessionID, input.CWD, msg)
	if err != nil {
		return err
	}

	msg = EnsureTableBlankLines(msg)

	ts := time.Now().Format("15:04:05")
	content := fmt.Sprintf("## Assistant (%s)\n\n%s\n\n", ts, msg)

	if err := appendToFile(notePath, content); err != nil {
		return err
	}

	// Cache the recorded message to avoid duplicates
	return os.WriteFile(lastMsgCachePath, []byte(msg), 0o644)
}
