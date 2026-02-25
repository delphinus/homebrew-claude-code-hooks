package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
			// No settings file yet → create with just hooks
			settings = map[string]interface{}{
				"hooks": hooks,
			}
			return writeSettings(settingsFile, settings, "created")
		}
		return fmt.Errorf("reading settings.json: %w", err)
	}

	if err := json.Unmarshal(settingsData, &settings); err != nil {
		return fmt.Errorf("parsing settings.json: %w", err)
	}

	// Merge: replace hooks key
	merged := make(map[string]interface{})
	for k, v := range settings {
		merged[k] = v
	}
	merged["hooks"] = hooks

	if diffMode {
		return showDiff(settingsData, merged)
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
