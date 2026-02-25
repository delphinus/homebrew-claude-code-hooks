package note

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
)

// AddPreviousLinks adds related links to an existing note for all other notes
// with the same session ID. Only adds links if the note doesn't already have
// a "related:" field in its frontmatter.
func AddPreviousLinks(notePath, sessionID, vaultDir string) error {
	// Find other notes with same session_id
	allNotes, err := FindNotesBySessionID(sessionID, vaultDir)
	if err != nil {
		return err
	}

	var prevNames []string
	for _, f := range allNotes {
		if f == notePath {
			continue
		}
		prevNames = append(prevNames, strings.TrimSuffix(filepath.Base(f), ".md"))
	}

	if len(prevNames) == 0 {
		return nil
	}

	content, err := os.ReadFile(notePath)
	if err != nil {
		return err
	}

	fm, body, err := frontmatter.Parse(string(content))
	if err != nil {
		return err
	}

	// Add related links to frontmatter
	for _, pn := range prevNames {
		link := fmt.Sprintf(`"[[%s]]"`, pn)
		if !containsString(fm.Related, link) {
			fm.Related = append(fm.Related, link)
		}
	}

	// Build new content
	var b strings.Builder
	b.WriteString(fm.Render())

	// Insert [!link] section before body
	b.WriteString("\n> [!link] Previous Sessions\n")
	for _, pn := range prevNames {
		fmt.Fprintf(&b, "> - [[%s]]\n", pn)
	}
	b.WriteString("\n")
	b.WriteString(body)

	return os.WriteFile(notePath, []byte(b.String()), 0o644)
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
