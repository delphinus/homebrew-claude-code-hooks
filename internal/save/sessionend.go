package save

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

// sessionEndSync controls whether SessionEnd runs inline (for tests)
// or spawns a detached background process (production).
var sessionEndSync bool

// handleSessionEnd spawns a detached background process to generate
// summaries and titles, then returns immediately so the hook exits
// before Claude Code kills it during shutdown.
func handleSessionEnd(input *hookdata.HookInput) error {
	cacheDir := config.CacheDir()
	sessionCachePath := filepath.Join(cacheDir, input.SessionID)

	if _, err := os.Stat(sessionCachePath); err != nil {
		return nil
	}

	if sessionEndSync {
		RunSessionEndBG(input.SessionID)
		return nil
	}

	// Spawn a detached background process to do the heavy work
	// (calling claude -p for summaries/titles). The hook returns
	// immediately so Claude Code won't kill us during shutdown.
	exe, err := os.Executable()
	if err != nil {
		return nil
	}

	cmd := exec.Command(exe, "_session-end-bg", input.SessionID)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return nil
	}

	// Detach: don't wait for the child
	cmd.Process.Release()
	return nil
}

// RunSessionEndBG is the background worker for SessionEnd processing.
// It is invoked as: claude-code-hooks _session-end-bg <session_id>
func RunSessionEndBG(sessionID string) {
	// Ignore signals so we survive even if parent is killed
	signal.Ignore(syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	cacheDir := config.CacheDir()
	sessionCachePath := filepath.Join(cacheDir, sessionID)
	vaultDir := config.VaultDir()

	sessionNotes, err := note.FindNotesBySessionID(sessionID, vaultDir)
	if err != nil || len(sessionNotes) == 0 {
		os.Remove(sessionCachePath)
		return
	}

	for _, notePath := range sessionNotes {
		if _, err := os.Stat(notePath); err != nil {
			continue
		}

		flushCachedPlan(sessionID, notePath)

		noteContent, err := readHead(notePath, 100000)
		if err != nil {
			continue
		}

		summary := claudeGenerate(
			"以下の Claude Code の会話ログを日本語で3〜5行で要約してください。要約のみを出力し、前置きは不要です。",
			noteContent,
		)

		if summary != "" {
			insertSummary(notePath, summary)
		}

		basename := strings.TrimSuffix(filepath.Base(notePath), ".md")
		parts := strings.SplitN(basename, "-", 4)
		if len(parts) < 4 {
			continue
		}
		prefix := strings.Join(parts[:3], "-")
		currentTitle := parts[3]
		project := filepath.Base(filepath.Dir(notePath))

		if currentTitle == project {
			newTitle := claudeGenerate(
				"以下の Claude Code の会話ログの内容を表す短いタイトル（50文字以内）を日本語で生成してください。タイトルのみを出力し、前置きや引用符は不要です。",
				noteContent,
			)

			if newTitle != "" {
				renameNote(notePath, prefix, newTitle, vaultDir)
			}
		}
	}

	os.Remove(sessionCachePath)
	os.Remove(filepath.Join(cacheDir, sessionID+"-last-msg"))
}

func readHead(path string, maxBytes int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	return string(buf[:n]), nil
}

func claudeGenerate(prompt, content string) string {
	cmd := exec.Command("claude", "-p", "--model", "haiku", "--setting-sources", "")
	cmd.Stdin = strings.NewReader(content)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Filter out CLAUDECODE to avoid "nested session" rejection.
	// --setting-sources "" prevents loading hooks config, so the inner
	// claude -p session won't fire its own SessionEnd hook.
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env

	cmd.Args = append(cmd.Args, prompt)

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func insertSummary(notePath, summary string) {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return
	}

	text := string(content)

	// Skip if summary already exists
	if strings.Contains(text, "> [!summary]") {
		return
	}

	// Find the end of frontmatter (second ---)
	idx := strings.Index(text, "---\n")
	if idx < 0 {
		return
	}
	rest := text[idx+4:]
	endFM := strings.Index(rest, "---\n")
	if endFM < 0 {
		return
	}

	// Position right after closing ---
	insertPos := idx + 4 + endFM + 4

	var b strings.Builder
	b.WriteString(text[:insertPos])
	b.WriteString("\n> [!summary]\n")
	for _, line := range strings.Split(summary, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	b.WriteString("\n")
	b.WriteString(text[insertPos:])

	os.WriteFile(notePath, []byte(b.String()), 0o644)
}

func renameNote(notePath, prefix, newTitle, vaultDir string) {
	oldBasename := strings.TrimSuffix(filepath.Base(notePath), ".md")

	fileTitle := note.MakeFilenameTitle(newTitle)
	displayTitle := note.MakeDisplayTitle(newTitle)
	idSlug := note.MakeIDSlug(fileTitle)

	if fileTitle == "" {
		return
	}

	newNotePath := filepath.Join(filepath.Dir(notePath), prefix+"-"+fileTitle+".md")
	newID := prefix + "-" + idSlug
	newBasename := prefix + "-" + fileTitle

	// Update frontmatter id and aliases
	content, err := os.ReadFile(notePath)
	if err != nil {
		return
	}

	text := string(content)

	// Update id field
	fm, body, err := frontmatter.Parse(text)
	if err != nil {
		return
	}

	fm.ID = newID
	if len(fm.Aliases) > 0 {
		fm.Aliases[0] = displayTitle
	} else {
		fm.Aliases = []string{displayTitle}
	}

	os.WriteFile(notePath, []byte(fm.Render()+body), 0o644)

	// Rename the file
	if err := os.Rename(notePath, newNotePath); err != nil {
		return
	}

	// Update references in other files
	updateReferences(oldBasename, newBasename, newNotePath, vaultDir)
}

func updateReferences(oldName, newName, excludePath, vaultDir string) {
	oldLink := "[[" + oldName + "]]"
	newLink := "[[" + newName + "]]"

	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == excludePath || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		text := string(content)
		if !strings.Contains(text, oldLink) {
			return nil
		}

		newText := strings.ReplaceAll(text, oldLink, newLink)
		os.WriteFile(path, []byte(newText), 0o644)
		return nil
	})
}
