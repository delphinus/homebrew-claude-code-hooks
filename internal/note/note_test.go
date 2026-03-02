package note

import (
	"os"
	"path/filepath"
	"testing"
)

func TestObsidianURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		want     string
	}{
		{
			name: "simple path",
			path: "/Users/test/vault/note.md",
			want: "obsidian://open?path=%2FUsers%2Ftest%2Fvault%2Fnote.md",
		},
		{
			name: "path with spaces uses %20 not +",
			path: "/Users/test/Library/Mobile Documents/iCloud~md~obsidian/Documents/Notes/Claude Code/note.md",
			want: "obsidian://open?path=%2FUsers%2Ftest%2FLibrary%2FMobile%20Documents%2FiCloud~md~obsidian%2FDocuments%2FNotes%2FClaude%20Code%2Fnote.md",
		},
		{
			name: "path with Japanese characters",
			path: "/Users/test/vault/テストノート.md",
			want: "obsidian://open?path=%2FUsers%2Ftest%2Fvault%2F%E3%83%86%E3%82%B9%E3%83%88%E3%83%8E%E3%83%BC%E3%83%88.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := obsidianURL(tt.path)
			if got != tt.want {
				t.Errorf("obsidianURL(%q) =\n  %s\nwant:\n  %s", tt.path, got, tt.want)
			}
		})
	}
}

func TestRepoRoot(t *testing.T) {
	t.Run("finds git repo root from subdirectory", func(t *testing.T) {
		// Create a temp directory simulating a repo with subdirectories
		root := t.TempDir()
		gitDir := filepath.Join(root, ".git")
		if err := os.Mkdir(gitDir, 0o755); err != nil {
			t.Fatal(err)
		}
		sub := filepath.Join(root, "src", "pkg")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}

		got := repoRoot(sub)
		if got != root {
			t.Errorf("repoRoot(%q) = %q, want %q", sub, got, root)
		}
	})

	t.Run("finds git repo root at current directory", func(t *testing.T) {
		root := t.TempDir()
		gitDir := filepath.Join(root, ".git")
		if err := os.Mkdir(gitDir, 0o755); err != nil {
			t.Fatal(err)
		}

		got := repoRoot(root)
		if got != root {
			t.Errorf("repoRoot(%q) = %q, want %q", root, got, root)
		}
	})

	t.Run("handles git worktree (.git file)", func(t *testing.T) {
		root := t.TempDir()
		// Worktrees have a .git file instead of a .git directory
		gitFile := filepath.Join(root, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o644); err != nil {
			t.Fatal(err)
		}
		sub := filepath.Join(root, "internal")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}

		got := repoRoot(sub)
		if got != root {
			t.Errorf("repoRoot(%q) = %q, want %q", sub, got, root)
		}
	})

	t.Run("stops at submodule root, not parent repo", func(t *testing.T) {
		// Simulate: parent-repo/.git (dir) + parent-repo/libs/sub-module/.git (file)
		parentRepo := t.TempDir()
		if err := os.Mkdir(filepath.Join(parentRepo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		submoduleRoot := filepath.Join(parentRepo, "libs", "sub-module")
		if err := os.MkdirAll(submoduleRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		// Submodules have a .git file pointing to parent's .git/modules/
		gitFile := filepath.Join(submoduleRoot, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../../.git/modules/libs/sub-module"), 0o644); err != nil {
			t.Fatal(err)
		}
		sub := filepath.Join(submoduleRoot, "src", "pkg")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}

		got := repoRoot(sub)
		if got != submoduleRoot {
			t.Errorf("repoRoot(%q) = %q, want %q (should stop at submodule, not parent)", sub, got, submoduleRoot)
		}
	})

	t.Run("returns original dir when no repo found", func(t *testing.T) {
		// Use a temp directory with no .git anywhere in it
		dir := t.TempDir()
		sub := filepath.Join(dir, "a", "b", "c")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}

		got := repoRoot(sub)
		if got != sub {
			t.Errorf("repoRoot(%q) = %q, want %q (should fall back to original dir)", sub, got, sub)
		}
	})
}
