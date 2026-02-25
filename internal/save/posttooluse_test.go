package save

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
)

// setupTestDirs creates temporary vault and cache directories, sets the
// corresponding environment variables, and returns a cleanup function.
func setupTestDirs(t *testing.T) (vaultDir, cacheDir string) {
	t.Helper()

	vaultDir = t.TempDir()
	cacheDir = t.TempDir()
	t.Setenv("CLAUDE_OBSIDIAN_VAULT", vaultDir)
	t.Setenv("CLAUDE_OBSIDIAN_CACHE", cacheDir)
	return vaultDir, cacheDir
}

func TestPlanMode_FullFlow(t *testing.T) {
	vaultDir, cacheDir := setupTestDirs(t)

	sessionID := "test-session-1234"
	cwd := "/tmp/test-project"

	// Step 1: EnterPlanMode — should create a flag file and append callout to note
	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}

	if err := os.MkdirAll(filepath.Join(vaultDir, "test-project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	// Verify flag file was created
	flagPath := filepath.Join(cacheDir, sessionID+"-in-plan")
	if _, err := os.Stat(flagPath); err != nil {
		t.Fatalf("plan flag file not created: %v", err)
	}

	// Verify note exists and contains plan mode callout
	noteContent := readNoteContent(t, vaultDir)
	if !strings.Contains(noteContent, "[!plan] Entering Plan Mode") {
		t.Errorf("note should contain plan mode callout, got:\n%s", noteContent)
	}

	// Step 2: Write during plan mode — should cache the plan content
	planContent := "# Implementation Plan\n\n## Steps\n\n1. First step\n2. Second step\n3. Third step\n"
	writeInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/.claude/plan.md",
			Content:  planContent,
		},
	}

	if err := Run(writeInput); err != nil {
		t.Fatalf("Write during plan mode failed: %v", err)
	}

	// Verify plan was cached
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	cachedData, err := os.ReadFile(planCachePath)
	if err != nil {
		t.Fatalf("plan cache file not created: %v", err)
	}
	if string(cachedData) != planContent {
		t.Errorf("cached plan content mismatch.\nwant: %q\ngot:  %q", planContent, string(cachedData))
	}

	// Step 3: ExitPlanMode — should read cache and append plan to note
	exitInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "ExitPlanMode",
	}

	if err := Run(exitInput); err != nil {
		t.Fatalf("ExitPlanMode failed: %v", err)
	}

	// Verify plan was appended to note
	noteContent = readNoteContent(t, vaultDir)
	if !strings.Contains(noteContent, "[!plan]- Implementation Plan") {
		t.Errorf("note should contain plan callout with title, got:\n%s", noteContent)
	}
	if !strings.Contains(noteContent, "> # Implementation Plan") {
		t.Errorf("note should contain plan heading, got:\n%s", noteContent)
	}
	if !strings.Contains(noteContent, "> 1. First step") {
		t.Errorf("note should contain plan steps, got:\n%s", noteContent)
	}

	// Verify cache files were cleaned up
	if _, err := os.Stat(planCachePath); !os.IsNotExist(err) {
		t.Error("plan cache file should have been removed after ExitPlanMode")
	}
	if _, err := os.Stat(flagPath); !os.IsNotExist(err) {
		t.Error("plan flag file should have been removed after ExitPlanMode")
	}
}

func TestPlanMode_ExitWithoutPlan(t *testing.T) {
	vaultDir, cacheDir := setupTestDirs(t)

	sessionID := "test-session-no-plan"
	cwd := "/tmp/test-project"

	// Enter plan mode
	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}

	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	// Exit plan mode without writing anything
	exitInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "ExitPlanMode",
	}

	if err := Run(exitInput); err != nil {
		t.Fatalf("ExitPlanMode failed: %v", err)
	}

	// Note should only contain the "Entering Plan Mode" callout, not a plan
	noteContent := readNoteContent(t, vaultDir)
	if strings.Contains(noteContent, "[!plan]-") {
		t.Errorf("note should not contain collapsible plan callout when no plan was written, got:\n%s", noteContent)
	}

	// Flag should be cleaned up
	flagPath := filepath.Join(cacheDir, sessionID+"-in-plan")
	if _, err := os.Stat(flagPath); !os.IsNotExist(err) {
		t.Error("plan flag file should have been removed")
	}
}

func TestPlanMode_WriteOutsidePlanMode(t *testing.T) {
	_, cacheDir := setupTestDirs(t)

	sessionID := "test-session-outside"
	cwd := "/tmp/test-project"

	// Write without entering plan mode
	writeInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/main.go",
			Content:  "package main\n",
		},
	}

	if err := Run(writeInput); err != nil {
		t.Fatalf("Write outside plan mode failed: %v", err)
	}

	// Plan cache should NOT be created
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	if _, err := os.Stat(planCachePath); !os.IsNotExist(err) {
		t.Error("plan cache file should not be created when not in plan mode")
	}
}

func TestPlanMode_MultipleWrites(t *testing.T) {
	_, cacheDir := setupTestDirs(t)

	sessionID := "test-session-multi"
	cwd := "/tmp/test-project"

	// Enter plan mode
	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}

	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	// First write
	write1 := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/.claude/plan.md",
			Content:  "# Draft Plan\nold content",
		},
	}
	if err := Run(write1); err != nil {
		t.Fatalf("first Write failed: %v", err)
	}

	// Second write (updated plan) — should overwrite
	finalPlan := "# Final Plan\nnew content"
	write2 := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/.claude/plan.md",
			Content:  finalPlan,
		},
	}
	if err := Run(write2); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	// Verify only the final plan is cached
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	cachedData, err := os.ReadFile(planCachePath)
	if err != nil {
		t.Fatalf("plan cache file not found: %v", err)
	}
	if string(cachedData) != finalPlan {
		t.Errorf("cached plan should be the final version.\nwant: %q\ngot:  %q", finalPlan, string(cachedData))
	}
}

func TestPlanMode_EditCachesPlan(t *testing.T) {
	_, cacheDir := setupTestDirs(t)

	sessionID := "test-session-edit"
	cwd := "/tmp/test-project"

	// Enter plan mode
	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}
	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	// Create a plan file on disk (simulating an existing file)
	planDir := filepath.Join(cwd, ".claude", "plans")
	_ = os.MkdirAll(planDir, 0o755)
	planFilePath := filepath.Join(planDir, "test-plan.md")
	planContent := "# Edited Plan\n\n1. Step A\n2. Step B\n"
	_ = os.WriteFile(planFilePath, []byte(planContent), 0o644)

	// Edit the plan file during plan mode
	editInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Edit",
		ToolInput: hookdata.ToolInput{
			FilePath: planFilePath,
		},
	}
	if err := Run(editInput); err != nil {
		t.Fatalf("Edit during plan mode failed: %v", err)
	}

	// Verify plan was cached by reading the file from disk
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	cachedData, err := os.ReadFile(planCachePath)
	if err != nil {
		t.Fatalf("plan cache file not created for Edit: %v", err)
	}
	if string(cachedData) != planContent {
		t.Errorf("cached plan content mismatch.\nwant: %q\ngot:  %q", planContent, string(cachedData))
	}
}

func TestPlanMode_SessionEndFlushesOrphanedPlan(t *testing.T) {
	vaultDir, cacheDir := setupTestDirs(t)

	sessionID := "test-session-orphan"
	cwd := "/tmp/test-project"

	// Step 1: EnterPlanMode
	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}
	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	// Step 2: Write plan content
	planContent := "# Orphaned Plan\n\n## This plan was never exited\n\n1. Step one\n2. Step two\n"
	writeInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/.claude/plan.md",
			Content:  planContent,
		},
	}
	if err := Run(writeInput); err != nil {
		t.Fatalf("Write during plan mode failed: %v", err)
	}

	// Verify plan was cached
	planCachePath := filepath.Join(cacheDir, sessionID+"-plan")
	if _, err := os.Stat(planCachePath); err != nil {
		t.Fatalf("plan cache file not created: %v", err)
	}

	// Step 3: SessionEnd fires WITHOUT ExitPlanMode
	// (simulates user rejecting ExitPlanMode or session ending abruptly)
	sessionEndInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "SessionEnd",
		CWD:           cwd,
	}
	if err := Run(sessionEndInput); err != nil {
		t.Fatalf("SessionEnd failed: %v", err)
	}

	// Verify plan was flushed to note
	noteContent := readNoteContent(t, vaultDir)
	if !strings.Contains(noteContent, "[!plan]- Orphaned Plan") {
		t.Errorf("SessionEnd should have flushed orphaned plan to note, got:\n%s", noteContent)
	}
	if !strings.Contains(noteContent, "> 1. Step one") {
		t.Errorf("note should contain plan steps, got:\n%s", noteContent)
	}

	// Verify cache files were cleaned up
	if _, err := os.Stat(planCachePath); !os.IsNotExist(err) {
		t.Error("plan cache file should have been removed after SessionEnd")
	}
	flagPath := filepath.Join(cacheDir, sessionID+"-in-plan")
	if _, err := os.Stat(flagPath); !os.IsNotExist(err) {
		t.Error("plan flag file should have been removed after SessionEnd")
	}
}

func TestPlanMode_RepostOnSessionResume(t *testing.T) {
	vaultDir, cacheDir := setupTestDirs(t)

	sessionID := "test-session-repost"
	cwd := "/tmp/test-project"

	// --- Session 1: Enter plan mode, write plan, session ends ---

	enterInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "EnterPlanMode",
	}
	if err := Run(enterInput); err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	planContent := "# My Plan\n\n1. Do thing A\n2. Do thing B\n"
	writeInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "PostToolUse",
		CWD:           cwd,
		ToolName:      "Write",
		ToolInput: hookdata.ToolInput{
			FilePath: "/tmp/test-project/.claude/plan.md",
			Content:  planContent,
		},
	}
	if err := Run(writeInput); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// SessionEnd fires (ExitPlanMode never fired)
	sessionEndInput := &hookdata.HookInput{
		SessionID:     sessionID,
		HookEventName: "SessionEnd",
		CWD:           cwd,
	}
	if err := Run(sessionEndInput); err != nil {
		t.Fatalf("SessionEnd failed: %v", err)
	}

	// Verify plan-last cache was saved (keyed by project name)
	planLastPath := filepath.Join(cacheDir, "test-project-plan-last")
	if _, err := os.Stat(planLastPath); err != nil {
		t.Fatalf("plan-last cache should exist after SessionEnd: %v", err)
	}

	// --- Session 2: Different session ID, same project ---

	newSessionID := "different-session-5678"
	promptInput := &hookdata.HookInput{
		SessionID:     newSessionID,
		HookEventName: "UserPromptSubmit",
		CWD:           cwd,
		Prompt:        "続きをお願いします",
	}
	if err := Run(promptInput); err != nil {
		t.Fatalf("UserPromptSubmit failed: %v", err)
	}

	// Find the NEW note (created for the new session)
	noteContent := readNoteBySessionID(t, vaultDir, newSessionID)

	// Verify plan was re-posted with non-collapsible callout
	if !strings.Contains(noteContent, "[!plan] My Plan") {
		t.Errorf("new session note should contain non-collapsible plan callout, got:\n%s", noteContent)
	}
	// Verify it's NOT collapsible (no dash after [!plan])
	if strings.Contains(noteContent, "[!plan]- My Plan") {
		t.Errorf("re-posted plan should NOT be collapsible, got:\n%s", noteContent)
	}
	if !strings.Contains(noteContent, "> 1. Do thing A") {
		t.Errorf("re-posted plan should contain plan steps, got:\n%s", noteContent)
	}

	// Verify plan-last cache was cleaned up
	if _, err := os.Stat(planLastPath); !os.IsNotExist(err) {
		t.Error("plan-last cache should have been removed after re-posting")
	}

	// Verify link to original note was added
	if !strings.Contains(noteContent, "[!link] Plan from") {
		t.Errorf("new session note should have a 'Plan from' link to the original note, got:\n%s", noteContent)
	}
}

func TestPlanMode_StaleRepostIgnored(t *testing.T) {
	vaultDir, cacheDir := setupTestDirs(t)

	cwd := "/tmp/test-project"

	// Manually create a stale plan-last cache (older than planRepostMaxAge)
	planLastPath := filepath.Join(cacheDir, "test-project-plan-last")
	_ = os.WriteFile(planLastPath, []byte("# Old Plan\n\nThis is stale\n"), 0o644)

	// Backdate the file by 5 minutes
	staleTime := time.Now().Add(-5 * time.Minute)
	_ = os.Chtimes(planLastPath, staleTime, staleTime)

	// New session in the same project
	promptInput := &hookdata.HookInput{
		SessionID:     "unrelated-session",
		HookEventName: "UserPromptSubmit",
		CWD:           cwd,
		Prompt:        "何か別の作業",
	}
	if err := Run(promptInput); err != nil {
		t.Fatalf("UserPromptSubmit failed: %v", err)
	}

	// Note should NOT contain the stale plan
	noteContent := readNoteContent(t, vaultDir)
	if strings.Contains(noteContent, "[!plan]") {
		t.Errorf("stale plan should NOT be re-posted, got:\n%s", noteContent)
	}

	// Stale cache file should have been deleted
	if _, err := os.Stat(planLastPath); !os.IsNotExist(err) {
		t.Error("stale plan-last cache should have been removed")
	}
}

// readNoteContent finds and reads the first .md file in the vault directory.
func readNoteContent(t *testing.T, vaultDir string) string {
	t.Helper()
	notes := findAllNotes(t, vaultDir)
	if len(notes) == 0 {
		t.Fatal("no .md note file found in vault dir")
	}
	data, err := os.ReadFile(notes[0])
	if err != nil {
		t.Fatalf("reading note file: %v", err)
	}
	return string(data)
}

// readLatestNoteContent finds and reads the last .md file (by path order) in the vault directory.
func readLatestNoteContent(t *testing.T, vaultDir string) string {
	t.Helper()
	notes := findAllNotes(t, vaultDir)
	if len(notes) == 0 {
		t.Fatal("no .md note file found in vault dir")
	}
	data, err := os.ReadFile(notes[len(notes)-1])
	if err != nil {
		t.Fatalf("reading note file: %v", err)
	}
	return string(data)
}

// readNoteBySessionID finds and reads the .md file containing the given session_id.
func readNoteBySessionID(t *testing.T, vaultDir, sessionID string) string {
	t.Helper()
	target := "session_id: " + sessionID
	for _, notePath := range findAllNotes(t, vaultDir) {
		data, err := os.ReadFile(notePath)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), target) {
			return string(data)
		}
	}
	t.Fatalf("no note found for session_id %s", sessionID)
	return ""
}

func findAllNotes(t *testing.T, vaultDir string) []string {
	t.Helper()
	var notes []string
	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			notes = append(notes, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking vault dir: %v", err)
	}
	return notes
}
