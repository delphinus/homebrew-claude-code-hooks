package config

import (
	"os"
	"path/filepath"
)

const (
	defaultVaultSubdir = "Library/Mobile Documents/iCloud~md~obsidian/Documents/Notes/Claude Code"
	defaultCacheSubdir = ".cache/claude-obsidian"
)

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// VaultDir returns the Obsidian vault directory for Claude Code notes.
// Reads CLAUDE_OBSIDIAN_VAULT env var, falling back to the default iCloud path.
func VaultDir() string {
	if v := os.Getenv("CLAUDE_OBSIDIAN_VAULT"); v != "" {
		return expandHome(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultVaultSubdir)
}

// CacheDir returns the cache directory for session state.
func CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultCacheSubdir)
}
