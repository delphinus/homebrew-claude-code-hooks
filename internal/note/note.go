package note

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
)

// SessionCache holds the cached state for a session.
type SessionCache struct {
	NotePath string
	CWD      string
}

// LoadSessionCache reads the session cache file.
// Returns nil if the cache file doesn't exist.
func LoadSessionCache(sessionID string) (*SessionCache, error) {
	path := filepath.Join(config.CacheDir(), sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.SplitN(string(data), "\n", 2)
	sc := &SessionCache{NotePath: lines[0]}
	if len(lines) > 1 {
		sc.CWD = lines[1]
	}
	return sc, nil
}

// SaveSessionCache writes the session cache file.
func SaveSessionCache(sessionID, notePath, cwd string) error {
	dir := config.CacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, sessionID)
	return os.WriteFile(path, []byte(notePath+"\n"+cwd), 0o644)
}

// GetOrCreateNote returns the path to the note for the current session,
// creating a new note if necessary. If the CWD has changed, a new note is created
// and the old note gets a "Continued in" link.
func GetOrCreateNote(sessionID, cwd, prompt string) (string, error) {
	vaultDir := config.VaultDir()

	cache, err := LoadSessionCache(sessionID)
	if err != nil {
		return "", fmt.Errorf("loading session cache: %w", err)
	}

	var oldNote string
	if cache != nil {
		// CWD unchanged or unknown → return existing note
		if cwd == "" || cache.CWD == "" || cwd == cache.CWD {
			return cache.NotePath, nil
		}
		// CWD changed → will create new note, keep reference to old
		oldNote = cache.NotePath
	}

	project := "unknown"
	if cwd != "" {
		project = filepath.Base(repoRoot(cwd))
	}

	now := time.Now()
	ts := now.Format("20060102-150405")

	displayTitle := MakeDisplayTitle(prompt)
	if displayTitle == "" {
		displayTitle = project
	}

	fileTitle := MakeFilenameTitle(prompt)
	if fileTitle == "" {
		fileTitle = project
	}

	idSlug := MakeIDSlug(fileTitle)
	sidShort := sessionID
	if len(sidShort) > 4 {
		sidShort = sidShort[:4]
	}

	noteID := fmt.Sprintf("%s-%s-%s", ts, sidShort, idSlug)
	projectDir := filepath.Join(vaultDir, project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", fmt.Errorf("creating project directory: %w", err)
	}

	notePath := filepath.Join(projectDir, fmt.Sprintf("%s-%s-%s.md", ts, sidShort, fileTitle))

	// Save cache
	if err := SaveSessionCache(sessionID, notePath, cwd); err != nil {
		return "", fmt.Errorf("saving session cache: %w", err)
	}

	// Find existing notes with same session_id (for resumed sessions / project switches)
	prevNotes, _ := FindNotesBySessionID(sessionID, vaultDir)
	var prevNames []string
	for _, pn := range prevNotes {
		if pn == notePath {
			continue
		}
		prevNames = append(prevNames, strings.TrimSuffix(filepath.Base(pn), ".md"))
	}

	// Build frontmatter
	fm := &frontmatter.Frontmatter{
		ID:            noteID,
		Aliases:       []string{displayTitle},
		Tags:          []string{"claude-code"},
		Date:          now.Format("2006-01-02T15:04:05"),
		SessionID:     sessionID,
		Hostname:      hostname(),
		CWD:           cwd,
		ClaudeVersion: claudeVersion(),
	}
	if len(prevNames) > 0 {
		for _, pn := range prevNames {
			fm.Related = append(fm.Related, fmt.Sprintf(`"[[%s]]"`, pn))
		}
	}

	// Build note content
	var b strings.Builder
	b.WriteString(fm.Render())
	b.WriteString("\n")

	if oldNote != "" {
		if _, err := os.Stat(oldNote); err == nil {
			oldName := strings.TrimSuffix(filepath.Base(oldNote), ".md")
			fmt.Fprintf(&b, "> [!link] Continued from [[%s]]\n\n", oldName)
		}
	} else if len(prevNames) > 0 {
		b.WriteString("> [!link] Previous Sessions\n")
		for _, pn := range prevNames {
			fmt.Fprintf(&b, "> - [[%s]]\n", pn)
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(notePath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing note: %w", err)
	}

	openInObsidian(notePath)

	// Add "Continued in" link to old note
	if oldNote != "" {
		if _, err := os.Stat(oldNote); err == nil {
			newName := strings.TrimSuffix(filepath.Base(notePath), ".md")
			f, err := os.OpenFile(oldNote, os.O_APPEND|os.O_WRONLY, 0o644)
			if err == nil {
				fmt.Fprintf(f, "> [!link] Continued in [[%s]]\n\n", newName)
				f.Close()
			}
		}
	}

	return notePath, nil
}

// FindNotesBySessionID searches all .md files in vaultDir for those containing
// the given session_id in their frontmatter.
func FindNotesBySessionID(sessionID, vaultDir string) ([]string, error) {
	var results []string
	target := "session_id: " + sessionID

	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == target {
				results = append(results, path)
				break
			}
			// Stop scanning after frontmatter ends (second ---)
			if line == "---" && len(results) == 0 {
				// Could be opening or closing ---, keep scanning through frontmatter
			}
		}
		return nil
	})
	return results, err
}

// repoRoot walks up from dir looking for a .git directory or file.
// Returns the repository root if found, otherwise returns dir unchanged.
func repoRoot(dir string) string {
	cur := dir
	for {
		if fi, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			// .git can be a directory (normal repo) or a file (worktree)
			if fi.IsDir() || fi.Mode().IsRegular() {
				return cur
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			// Reached filesystem root without finding .git
			return dir
		}
		cur = parent
	}
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func claudeVersion() string {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// vaultRoot walks up from dir looking for an .obsidian directory.
// Returns the vault root if found, otherwise returns "".
func vaultRoot(dir string) string {
	cur := dir
	for {
		if fi, err := os.Stat(filepath.Join(cur, ".obsidian")); err == nil && fi.IsDir() {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

// hasAdvancedURI checks whether the Advanced URI plugin is installed in the vault.
func hasAdvancedURI(root string) bool {
	fi, err := os.Stat(filepath.Join(root, ".obsidian", "plugins", "obsidian-advanced-uri"))
	return err == nil && fi.IsDir()
}

// obsidianOpenURL builds a basic obsidian://open URI.
func obsidianOpenURL(notePath string) string {
	return "obsidian://open?path=" + url.PathEscape(notePath)
}

// advancedURI builds an obsidian://advanced-uri URI with openmode=tab.
func advancedURI(vaultName, relPath string) string {
	return "obsidian://advanced-uri?vault=" + url.PathEscape(vaultName) +
		"&filepath=" + url.PathEscape(relPath) +
		"&openmode=tab"
}

// openInObsidian opens the given note in Obsidian.
// Skips if the file is already open in Obsidian (checks workspace.json).
// Uses Advanced URI plugin (new tab) if available, otherwise falls back to obsidian://open.
// Closes old tabs via REST API when the tracked count exceeds maxTabs.
func openInObsidian(notePath string) {
	root := vaultRoot(filepath.Dir(notePath))
	if root == "" {
		return
	}

	if isFileOpenInObsidian(root, notePath) {
		trackTab(notePath)
		return
	}
	uri := obsidianOpenURL(notePath)
	if hasAdvancedURI(root) {
		vaultName := filepath.Base(root)
		if relPath, err := filepath.Rel(root, notePath); err == nil {
			uri = advancedURI(vaultName, relPath)
		}
	}
	exec.Command("open", "-g", uri).Start()

	toClose := trackTab(notePath)
	if len(toClose) > 0 {
		// Check REST API and workspace state synchronously (so warnings are visible)
		cfg, relPaths := prepareCloseOldTabs(root, toClose)
		if cfg != nil && len(relPaths) > 0 {
			closeTabsAsync(cfg, relPaths)
		}
	}
}

