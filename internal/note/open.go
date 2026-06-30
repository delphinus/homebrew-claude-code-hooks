package note

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
)

// NoteMeta describes a session note for listing/selection.
type NoteMeta struct {
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	SessionID string    `json:"session_id"`
	CWD       string    `json:"cwd"`
	Project   string    `json:"project"`
	Date      string    `json:"date"`
	Modified  time.Time `json:"modified"`
}

// parseNoteFrontmatter reads only the frontmatter block of a note file.
// It stops at the closing "---" so large notes aren't read in full.
func parseNoteFrontmatter(path string) (*frontmatter.Frontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return nil, fmt.Errorf("no frontmatter in %s", path)
	}
	var b strings.Builder
	b.WriteString("---\n")
	for sc.Scan() {
		line := sc.Text()
		b.WriteString(line)
		b.WriteString("\n")
		if strings.TrimSpace(line) == "---" {
			fm, _, err := frontmatter.Parse(b.String())
			return fm, err
		}
	}
	return nil, fmt.Errorf("unclosed frontmatter in %s", path)
}

// noteTitle derives a human-readable title from frontmatter aliases,
// falling back to the filename with the leading timestamp/session prefix stripped.
func noteTitle(fm *frontmatter.Frontmatter, path string) string {
	if fm != nil && len(fm.Aliases) > 0 && fm.Aliases[0] != "" {
		return fm.Aliases[0]
	}
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	// filenames look like "20060102-150405-xxxx-title"; drop the 3 leading fields
	parts := strings.SplitN(base, "-", 4)
	if len(parts) == 4 {
		return parts[3]
	}
	return base
}

// ListNotes scans the vault and returns metadata for every session note,
// newest (by modification time) first.
func ListNotes() ([]NoteMeta, error) {
	vaultDir := config.VaultDir()
	if vaultDir == "" {
		return nil, fmt.Errorf("vault directory not resolved")
	}

	var metas []NoteMeta
	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		fm, ferr := parseNoteFrontmatter(path)
		if ferr != nil || fm == nil || fm.SessionID == "" {
			return nil // only list notes created by this tool
		}
		project := ""
		if rel, rerr := filepath.Rel(vaultDir, filepath.Dir(path)); rerr == nil && rel != "." {
			project = rel
		}
		metas = append(metas, NoteMeta{
			Path:      path,
			Title:     noteTitle(fm, path),
			SessionID: fm.SessionID,
			CWD:       fm.CWD,
			Project:   project,
			Date:      fm.Date,
			Modified:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Modified.After(metas[j].Modified)
	})
	return metas, nil
}

// RecentNotes returns the most recently modified session notes, up to limit.
func RecentNotes(limit int) ([]NoteMeta, error) {
	metas, err := ListNotes()
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(metas) > limit {
		metas = metas[:limit]
	}
	return metas, nil
}

// MostRecentNote returns the path of the most recently modified session note.
// This is the note for the currently active session, since it is appended to
// on every tool use.
func MostRecentNote() (string, error) {
	metas, err := ListNotes()
	if err != nil {
		return "", err
	}
	if len(metas) == 0 {
		return "", fmt.Errorf("no session notes found")
	}
	return metas[0].Path, nil
}

// ResolveBySessionID finds the note path for a session ID. It tries the session
// cache first, then matches notes by exact session_id, then by prefix (so a
// short id works). When several notes match, the most recent one is returned.
func ResolveBySessionID(id string) (string, error) {
	if sc, _ := LoadSessionCache(id); sc != nil && sc.NotePath != "" {
		if _, err := os.Stat(sc.NotePath); err == nil {
			return sc.NotePath, nil
		}
	}

	metas, err := ListNotes()
	if err != nil {
		return "", err
	}
	// metas is already newest-first, so the first match wins.
	for _, m := range metas {
		if m.SessionID == id {
			return m.Path, nil
		}
	}
	for _, m := range metas {
		if strings.HasPrefix(m.SessionID, id) {
			return m.Path, nil
		}
	}
	return "", fmt.Errorf("no note found for session %q", id)
}

// OpenNote opens the given note in Obsidian, bringing the app to the front.
// Uses the Advanced URI plugin (new tab) when available, otherwise obsidian://open.
// Unlike the automatic open, it neither tracks nor closes tabs.
func OpenNote(notePath string) error {
	uri := obsidianOpenURL(notePath)
	if root := vaultRoot(filepath.Dir(notePath)); root != "" && hasAdvancedURI(root) {
		if relPath, err := filepath.Rel(root, notePath); err == nil {
			uri = advancedURI(filepath.Base(root), relPath)
		}
	}
	return exec.Command("open", uri).Start()
}
