package note

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeNote creates a note file with frontmatter and sets its mod time.
func writeNote(t *testing.T, dir, name, sessionID, alias string, mod time.Time) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	content := "---\nid: x\naliases:\n  - " + alias +
		"\ntags:\n  - claude-code\nsession_id: " + sessionID +
		"\ncwd: /home/me/proj\n---\n\nbody\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
	return path
}

func setupVault(t *testing.T) (string, map[string]string) {
	t.Helper()
	vault := t.TempDir()
	t.Setenv("CLAUDE_OBSIDIAN_VAULT", vault)
	t.Setenv("CLAUDE_OBSIDIAN_CACHE", t.TempDir())

	base := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	paths := map[string]string{
		"old": writeNote(t, filepath.Join(vault, "projA"), "20260630-120000-aaaa-old.md", "aaaa1111", "Old note", base),
		"mid": writeNote(t, filepath.Join(vault, "projA"), "20260630-130000-bbbb-mid.md", "bbbb2222", "Mid note", base.Add(time.Hour)),
		"new": writeNote(t, filepath.Join(vault, "projB"), "20260630-140000-cccc-new.md", "cccc3333", "New note", base.Add(2*time.Hour)),
	}
	// a stray markdown file without our frontmatter must be ignored
	if err := os.WriteFile(filepath.Join(vault, "random.md"), []byte("# not ours\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return vault, paths
}

func TestListNotes_SortedNewestFirstAndFiltered(t *testing.T) {
	_, paths := setupVault(t)

	metas, err := ListNotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 3 {
		t.Fatalf("ListNotes() returned %d notes, want 3 (stray file should be skipped)", len(metas))
	}
	if metas[0].Path != paths["new"] || metas[2].Path != paths["old"] {
		t.Errorf("not sorted newest-first: got %s ... %s", metas[0].Path, metas[2].Path)
	}
	if metas[0].Title != "New note" {
		t.Errorf("Title = %q, want %q", metas[0].Title, "New note")
	}
	if metas[0].Project != "projB" {
		t.Errorf("Project = %q, want %q", metas[0].Project, "projB")
	}
}

func TestMostRecentNote(t *testing.T) {
	_, paths := setupVault(t)
	got, err := MostRecentNote()
	if err != nil {
		t.Fatal(err)
	}
	if got != paths["new"] {
		t.Errorf("MostRecentNote() = %q, want %q", got, paths["new"])
	}
}

func TestRecentNotes_Limit(t *testing.T) {
	setupVault(t)
	metas, err := RecentNotes(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 2 {
		t.Fatalf("RecentNotes(2) = %d notes, want 2", len(metas))
	}
}

func TestResolveBySessionID(t *testing.T) {
	_, paths := setupVault(t)

	t.Run("exact", func(t *testing.T) {
		got, err := ResolveBySessionID("bbbb2222")
		if err != nil {
			t.Fatal(err)
		}
		if got != paths["mid"] {
			t.Errorf("got %q, want %q", got, paths["mid"])
		}
	})

	t.Run("prefix", func(t *testing.T) {
		got, err := ResolveBySessionID("cccc")
		if err != nil {
			t.Fatal(err)
		}
		if got != paths["new"] {
			t.Errorf("got %q, want %q", got, paths["new"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := ResolveBySessionID("zzzz"); err == nil {
			t.Error("expected error for unknown session id")
		}
	})

	t.Run("cache fast path", func(t *testing.T) {
		if err := SaveSessionCache("dddd4444", paths["old"], "/home/me/proj"); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveBySessionID("dddd4444")
		if err != nil {
			t.Fatal(err)
		}
		if got != paths["old"] {
			t.Errorf("got %q, want %q", got, paths["old"])
		}
	})
}
