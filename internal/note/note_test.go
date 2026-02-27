package note

import (
	"os"
	"path/filepath"
	"testing"
)

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
