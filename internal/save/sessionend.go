package save

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

func handleSessionEnd(input *hookdata.HookInput) error {
	cacheDir := config.CacheDir()
	sessionCachePath := filepath.Join(cacheDir, input.SessionID)

	if _, err := os.Stat(sessionCachePath); err != nil {
		return nil
	}

	vaultDir := config.VaultDir()

	// Find all notes for this session
	sessionNotes, err := note.FindNotesBySessionID(input.SessionID, vaultDir)
	if err != nil || len(sessionNotes) == 0 {
		os.Remove(sessionCachePath)
		return nil
	}

	for _, notePath := range sessionNotes {
		if _, err := os.Stat(notePath); err != nil {
			continue
		}

		// Read note content (up to 100KB)
		noteContent, err := readHead(notePath, 100000)
		if err != nil {
			continue
		}

		// Generate summary using claude CLI
		summary := claudeGenerate(
			"以下の Claude Code の会話ログを日本語で3〜5行で要約してください。要約のみを出力し、前置きは不要です。",
			noteContent,
		)

		if summary != "" {
			insertSummary(notePath, summary)
		}

		// Check if title is just the project name (fallback value)
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

	// Clean up cache files
	os.Remove(sessionCachePath)
	os.Remove(filepath.Join(cacheDir, input.SessionID+"-plan"))
	os.Remove(filepath.Join(cacheDir, input.SessionID+"-in-plan"))

	return nil
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
	cmd := exec.Command("claude", "-p", "--model", "haiku")
	cmd.Stdin = strings.NewReader(content)
	cmd.Env = append(os.Environ(), "CLAUDE_OBSIDIAN_SAVING=1")

	// Pass prompt via args
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
