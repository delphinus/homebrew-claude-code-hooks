package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	t.Run("returns default path when env is unset", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_CACHE", "")
		got := CacheDir()
		want := filepath.Join(home, defaultCacheSubdir)
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_CACHE", "/tmp/custom-cache")
		got := CacheDir()
		if got != "/tmp/custom-cache" {
			t.Errorf("CacheDir() = %q, want %q", got, "/tmp/custom-cache")
		}
	})

	t.Run("expands tilde in env value", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_CACHE", "~/my-cache")
		got := CacheDir()
		want := filepath.Join(home, "my-cache")
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})
}

func TestVaultDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	t.Run("returns default path when env is unset", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_VAULT", "")
		got := VaultDir()
		want := filepath.Join(home, defaultVaultSubdir)
		if got != want {
			t.Errorf("VaultDir() = %q, want %q", got, want)
		}
	})

	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_VAULT", "/tmp/custom-vault")
		got := VaultDir()
		if got != "/tmp/custom-vault" {
			t.Errorf("VaultDir() = %q, want %q", got, "/tmp/custom-vault")
		}
	})

	t.Run("expands tilde in env value", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_VAULT", "~/my-vault")
		got := VaultDir()
		want := filepath.Join(home, "my-vault")
		if got != want {
			t.Errorf("VaultDir() = %q, want %q", got, want)
		}
	})
}
