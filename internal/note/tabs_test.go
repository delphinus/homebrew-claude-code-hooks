package note

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveTrackedTabs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_OBSIDIAN_CACHE", tmp)

	// Initially empty
	tabs := loadTrackedTabs()
	if len(tabs) != 0 {
		t.Errorf("loadTrackedTabs() = %v, want empty", tabs)
	}

	// Save and reload
	want := []string{"/vault/a.md", "/vault/b.md"}
	if err := saveTrackedTabs(want); err != nil {
		t.Fatal(err)
	}
	got := loadTrackedTabs()
	if len(got) != len(want) {
		t.Fatalf("loadTrackedTabs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("loadTrackedTabs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTrackTab(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_OBSIDIAN_CACHE", tmp)
	t.Setenv("CLAUDE_OBSIDIAN_MAX_TABS", "3")

	// Add 3 tabs
	trackTab("/vault/a.md")
	trackTab("/vault/b.md")
	trackTab("/vault/c.md")

	tabs := loadTrackedTabs()
	if len(tabs) != 3 {
		t.Fatalf("expected 3 tabs, got %d: %v", len(tabs), tabs)
	}

	// Adding 4th should close the oldest
	toClose := trackTab("/vault/d.md")
	if len(toClose) != 1 || toClose[0] != "/vault/a.md" {
		t.Errorf("toClose = %v, want [/vault/a.md]", toClose)
	}

	tabs = loadTrackedTabs()
	if len(tabs) != 3 {
		t.Fatalf("expected 3 tabs after trim, got %d: %v", len(tabs), tabs)
	}
	if tabs[0] != "/vault/b.md" {
		t.Errorf("tabs[0] = %q, want /vault/b.md", tabs[0])
	}
}

func TestTrackTab_Duplicate(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_OBSIDIAN_CACHE", tmp)
	t.Setenv("CLAUDE_OBSIDIAN_MAX_TABS", "3")

	trackTab("/vault/a.md")
	trackTab("/vault/b.md")
	trackTab("/vault/c.md")

	// Re-tracking an existing tab should move it to the end
	toClose := trackTab("/vault/a.md")
	if len(toClose) != 0 {
		t.Errorf("toClose = %v, want empty", toClose)
	}

	tabs := loadTrackedTabs()
	if len(tabs) != 3 {
		t.Fatalf("expected 3 tabs, got %d: %v", len(tabs), tabs)
	}
	if tabs[2] != "/vault/a.md" {
		t.Errorf("tabs[2] = %q, want /vault/a.md (should be moved to end)", tabs[2])
	}
}

func TestMaxTabs(t *testing.T) {
	t.Run("default is 5", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_MAX_TABS", "")
		if got := maxTabs(); got != 5 {
			t.Errorf("maxTabs() = %d, want 5", got)
		}
	})

	t.Run("custom", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_MAX_TABS", "5")
		if got := maxTabs(); got != 5 {
			t.Errorf("maxTabs() = %d, want 5", got)
		}
	})

	t.Run("invalid falls back to default", func(t *testing.T) {
		t.Setenv("CLAUDE_OBSIDIAN_MAX_TABS", "abc")
		if got := maxTabs(); got != defaultMaxTabs {
			t.Errorf("maxTabs() = %d, want %d", got, defaultMaxTabs)
		}
	})
}

func TestEncodeRelPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple", "project/note.md", "project/note.md"},
		{"spaces", "Claude Code/my note.md", "Claude%20Code/my%20note.md"},
		{"hash", "project/note#1.md", "project/note%231.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeRelPath(tt.path); got != tt.want {
				t.Errorf("encodeRelPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadRESTAPIConfig(t *testing.T) {
	t.Run("returns config when plugin is installed", func(t *testing.T) {
		root := t.TempDir()
		pluginDir := filepath.Join(root, ".obsidian", "plugins", "obsidian-local-rest-api")
		if err := os.MkdirAll(pluginDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "data.json"), []byte(`{"apiKey":"test-key","insecurePort":27123}`), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := loadRESTAPIConfig(root)
		if cfg == nil {
			t.Fatal("loadRESTAPIConfig() = nil, want config")
		}
		if cfg.APIKey != "test-key" {
			t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
		}
		if cfg.Port != 27123 {
			t.Errorf("Port = %d, want 27123", cfg.Port)
		}
	})

	t.Run("returns nil when plugin is not installed", func(t *testing.T) {
		root := t.TempDir()
		if cfg := loadRESTAPIConfig(root); cfg != nil {
			t.Errorf("loadRESTAPIConfig() = %+v, want nil", cfg)
		}
	})

	t.Run("defaults insecurePort to 27123", func(t *testing.T) {
		root := t.TempDir()
		pluginDir := filepath.Join(root, ".obsidian", "plugins", "obsidian-local-rest-api")
		if err := os.MkdirAll(pluginDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "data.json"), []byte(`{"apiKey":"key"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := loadRESTAPIConfig(root)
		if cfg == nil {
			t.Fatal("loadRESTAPIConfig() = nil")
		}
		if cfg.Port != 27123 {
			t.Errorf("Port = %d, want 27123", cfg.Port)
		}
	})
}

func TestOpenFilesInWorkspace(t *testing.T) {
	t.Run("parses workspace.json with multiple tabs", func(t *testing.T) {
		root := t.TempDir()
		obsDir := filepath.Join(root, ".obsidian")
		if err := os.MkdirAll(obsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		workspace := `{
  "main": {
    "id": "root",
    "type": "split",
    "children": [
      {
        "id": "tabs1",
        "type": "tabs",
        "children": [
          {
            "id": "leaf1",
            "type": "leaf",
            "state": {
              "type": "markdown",
              "state": {
                "file": "project/note-a.md",
                "mode": "source"
              }
            }
          },
          {
            "id": "leaf2",
            "type": "leaf",
            "state": {
              "type": "markdown",
              "state": {
                "file": "project/note-b.md",
                "mode": "source"
              }
            }
          }
        ]
      }
    ]
  },
  "left": {
    "id": "sidebar",
    "type": "split",
    "children": [
      {
        "id": "leaf3",
        "type": "leaf",
        "state": {
          "type": "file-explorer",
          "state": {}
        }
      }
    ]
  }
}`
		if err := os.WriteFile(filepath.Join(obsDir, "workspace.json"), []byte(workspace), 0o644); err != nil {
			t.Fatal(err)
		}

		files := openFilesInWorkspace(root)
		if files == nil {
			t.Fatal("openFilesInWorkspace() = nil")
		}
		if !files["project/note-a.md"] {
			t.Error("expected project/note-a.md to be open")
		}
		if !files["project/note-b.md"] {
			t.Error("expected project/note-b.md to be open")
		}
		if len(files) != 2 {
			t.Errorf("expected 2 open files, got %d: %v", len(files), files)
		}
	})

	t.Run("returns nil when workspace.json does not exist", func(t *testing.T) {
		root := t.TempDir()
		files := openFilesInWorkspace(root)
		if files != nil {
			t.Errorf("openFilesInWorkspace() = %v, want nil", files)
		}
	})
}

func TestIsFileOpenInObsidian(t *testing.T) {
	t.Run("uses workspace.json when available", func(t *testing.T) {
		root := t.TempDir()
		obsDir := filepath.Join(root, ".obsidian")
		if err := os.MkdirAll(obsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		workspace := `{
  "main": {
    "type": "split",
    "children": [
      {
        "type": "tabs",
        "children": [
          {
            "type": "leaf",
            "state": {
              "type": "markdown",
              "state": {
                "file": "project/open-note.md"
              }
            }
          }
        ]
      }
    ]
  }
}`
		if err := os.WriteFile(filepath.Join(obsDir, "workspace.json"), []byte(workspace), 0o644); err != nil {
			t.Fatal(err)
		}

		if !isFileOpenInObsidian(root, filepath.Join(root, "project", "open-note.md")) {
			t.Error("isFileOpenInObsidian(open-note.md) = false, want true")
		}
		if isFileOpenInObsidian(root, filepath.Join(root, "project", "closed-note.md")) {
			t.Error("isFileOpenInObsidian(closed-note.md) = true, want false")
		}
	})

	t.Run("falls back to tracking list when workspace.json is unavailable", func(t *testing.T) {
		root := t.TempDir()
		// No .obsidian/workspace.json
		if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
			t.Fatal(err)
		}

		tmp := t.TempDir()
		t.Setenv("CLAUDE_OBSIDIAN_CACHE", tmp)

		trackedPath := filepath.Join(root, "project", "tracked-note.md")
		untrackedPath := filepath.Join(root, "project", "untracked-note.md")

		_ = saveTrackedTabs([]string{trackedPath})

		if !isFileOpenInObsidian(root, trackedPath) {
			t.Error("isFileOpenInObsidian(tracked) = false, want true (fallback to tracking)")
		}
		if isFileOpenInObsidian(root, untrackedPath) {
			t.Error("isFileOpenInObsidian(untracked) = true, want false")
		}
	})
}
