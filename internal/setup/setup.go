package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ownCommandBinary is the executable name of the hooks installed by this tool.
// It is used to distinguish our own hooks from hooks added by the user or by
// other tools, so that setup only manages the former.
const ownCommandBinary = "claude-code-hooks"

// isOwnCommand reports whether a hook command was installed by this tool.
// It matches on the basename of the first token, so bare names, absolute
// paths, and quoted commands are all recognized (e.g. "claude-code-hooks save",
// "/opt/homebrew/bin/claude-code-hooks save", "'claude-code-hooks' save").
func isOwnCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	first := strings.Trim(fields[0], `"'`)
	return filepath.Base(first) == ownCommandBinary
}

// mergeHooks combines this tool's own hooks (from hooks.json, "own") into the
// existing hooks section of settings.json ("existing") without disturbing hooks
// added by the user or by other tools. Previously-installed own hooks are
// removed first so that re-running setup updates them cleanly, then the fresh
// own hooks are prepended per event (own hooks ran first historically).
func mergeHooks(existing, own interface{}) map[string]interface{} {
	result := map[string]interface{}{}

	// Keep every foreign hook from the existing settings, dropping only our own.
	if em, ok := existing.(map[string]interface{}); ok {
		for event, v := range em {
			entries, ok := v.([]interface{})
			if !ok {
				result[event] = v // preserve unexpected shapes verbatim
				continue
			}
			var kept []interface{}
			for _, e := range entries {
				entry, ok := e.(map[string]interface{})
				if !ok {
					kept = append(kept, e)
					continue
				}
				hooksArr, ok := entry["hooks"].([]interface{})
				if !ok {
					kept = append(kept, entry)
					continue
				}
				var keptHooks []interface{}
				for _, h := range hooksArr {
					if ho, ok := h.(map[string]interface{}); ok {
						if cmd, ok := ho["command"].(string); ok && isOwnCommand(cmd) {
							continue // drop a previously-installed own hook
						}
					}
					keptHooks = append(keptHooks, h)
				}
				if len(keptHooks) == 0 {
					continue // entry contained only own hooks → drop it
				}
				entry["hooks"] = keptHooks
				kept = append(kept, entry)
			}
			if len(kept) > 0 {
				result[event] = kept
			}
		}
	}

	// Prepend the fresh own hooks so they run before any foreign hooks.
	if om, ok := own.(map[string]interface{}); ok {
		for event, v := range om {
			ownEntries, ok := v.([]interface{})
			if !ok {
				continue
			}
			existingEntries, _ := result[event].([]interface{})
			merged := make([]interface{}, 0, len(ownEntries)+len(existingEntries))
			merged = append(merged, ownEntries...)
			merged = append(merged, existingEntries...)
			result[event] = merged
		}
	}

	return result
}

// backupSettings copies the existing settings file to "<path>.bak" before it is
// overwritten. It is a no-op when the file does not yet exist.
func backupSettings(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading settings for backup: %w", err)
	}
	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return fmt.Errorf("writing settings backup: %w", err)
	}
	return nil
}

// Run executes the setup subcommand.
// It merges hooks.json into ~/.claude/settings.json.
func Run(diffMode bool) error {
	hooksFile := findHooksFile()
	if hooksFile == "" {
		return fmt.Errorf("hooks.json が見つかりません")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	settingsFile := filepath.Join(home, ".claude", "settings.json")
	claudeDir := filepath.Join(home, ".claude")

	// Read hooks.json
	hooksData, err := os.ReadFile(hooksFile)
	if err != nil {
		return fmt.Errorf("reading hooks.json: %w", err)
	}

	var hooks interface{}
	if err := json.Unmarshal(hooksData, &hooks); err != nil {
		return fmt.Errorf("parsing hooks.json: %w", err)
	}

	// Ensure ~/.claude directory exists
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("creating .claude dir: %w", err)
	}

	// Read or initialize settings
	var settings map[string]interface{}

	settingsData, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No settings file yet → create with just our own hooks
			settings = map[string]interface{}{
				"hooks": mergeHooks(nil, hooks),
			}
			return writeSettings(settingsFile, settings, "created")
		}
		return fmt.Errorf("reading settings.json: %w", err)
	}

	if err := json.Unmarshal(settingsData, &settings); err != nil {
		return fmt.Errorf("parsing settings.json: %w", err)
	}

	// Merge our own hooks into the existing hooks section, preserving any hooks
	// added by the user or by other tools.
	merged := make(map[string]interface{})
	for k, v := range settings {
		merged[k] = v
	}
	merged["hooks"] = mergeHooks(settings["hooks"], hooks)

	if diffMode {
		return showDiff(settingsData, merged)
	}

	if err := backupSettings(settingsFile); err != nil {
		return err
	}

	return writeSettings(settingsFile, merged, "updated")
}

func findHooksFile() string {
	// Check HOMEBREW_PREFIX first
	prefix := os.Getenv("HOMEBREW_PREFIX")
	if prefix == "" {
		prefix = "/opt/homebrew"
	}

	path := filepath.Join(prefix, "share", "claude-code-hooks", "hooks.json")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Fallback: check relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(filepath.Dir(exe)) // up from bin/
		path = filepath.Join(dir, "share", "claude-code-hooks", "hooks.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func writeSettings(path string, data interface{}, action string) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	fmt.Printf("%s: %s\n", action, path)
	return nil
}

func showDiff(currentData []byte, merged interface{}) error {
	// Sort current JSON
	var current interface{}
	json.Unmarshal(currentData, &current)
	currentSorted, _ := json.MarshalIndent(current, "", "  ")

	// Sort merged JSON
	mergedSorted, _ := json.MarshalIndent(merged, "", "  ")

	// Write to temp files and diff
	tmpCurrent, err := os.CreateTemp("", "settings-current-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpCurrent.Name())
	tmpCurrent.Write(append(currentSorted, '\n'))
	tmpCurrent.Close()

	tmpMerged, err := os.CreateTemp("", "settings-merged-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpMerged.Name())
	tmpMerged.Write(append(mergedSorted, '\n'))
	tmpMerged.Close()

	cmd := exec.Command("diff", tmpCurrent.Name(), tmpMerged.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // diff exits 1 when files differ, which is expected

	return nil
}
