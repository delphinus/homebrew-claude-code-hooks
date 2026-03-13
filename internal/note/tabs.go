package note

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
)

const defaultMaxTabs = 5

func maxTabs() int {
	if v := os.Getenv("CLAUDE_OBSIDIAN_MAX_TABS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxTabs
}

func tabsFilePath() string {
	return filepath.Join(config.CacheDir(), "opened-tabs")
}

func loadTrackedTabs() []string {
	data, err := os.ReadFile(tabsFilePath())
	if err != nil {
		return nil
	}
	var tabs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tabs = append(tabs, line)
		}
	}
	return tabs
}

func saveTrackedTabs(tabs []string) error {
	dir := config.CacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var content string
	if len(tabs) > 0 {
		content = strings.Join(tabs, "\n") + "\n"
	}
	return os.WriteFile(tabsFilePath(), []byte(content), 0o644)
}

// trackTab adds notePath to the opened-tabs list.
// Returns paths that should be closed (those exceeding maxTabs).
func trackTab(notePath string) []string {
	tabs := loadTrackedTabs()

	// Remove if already present (to move to end)
	var updated []string
	for _, t := range tabs {
		if t != notePath {
			updated = append(updated, t)
		}
	}
	updated = append(updated, notePath)

	var toClose []string
	max := maxTabs()
	if len(updated) > max {
		toClose = updated[:len(updated)-max]
		updated = updated[len(updated)-max:]
	}

	_ = saveTrackedTabs(updated)
	return toClose
}

// openFilesInWorkspace reads .obsidian/workspace.json and returns
// the set of vault-relative file paths that are currently open as tabs.
func openFilesInWorkspace(root string) map[string]bool {
	data, err := os.ReadFile(filepath.Join(root, ".obsidian", "workspace.json"))
	if err != nil {
		return nil
	}
	var ws interface{}
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil
	}
	files := make(map[string]bool)
	extractOpenFiles(ws, files)
	return files
}

// extractOpenFiles recursively walks the workspace JSON tree and collects
// file paths from leaf states (state -> state -> file).
func extractOpenFiles(v interface{}, files map[string]bool) {
	switch val := v.(type) {
	case map[string]interface{}:
		if state, ok := val["state"].(map[string]interface{}); ok {
			if inner, ok := state["state"].(map[string]interface{}); ok {
				if file, ok := inner["file"].(string); ok && file != "" {
					files[file] = true
				}
			}
		}
		for _, child := range val {
			extractOpenFiles(child, files)
		}
	case []interface{}:
		for _, item := range val {
			extractOpenFiles(item, files)
		}
	}
}

// isFileOpenInObsidian checks whether a file is actually open in Obsidian.
// First checks workspace.json; if unavailable, falls back to the tracking list.
func isFileOpenInObsidian(root, notePath string) bool {
	relPath, err := filepath.Rel(root, notePath)
	if err != nil {
		return false
	}
	openFiles := openFilesInWorkspace(root)
	if openFiles != nil {
		return openFiles[relPath]
	}
	// Fallback: check tracking list when workspace.json is unavailable
	// (e.g. iCloud sync conflict, Obsidian not running)
	tabs := loadTrackedTabs()
	for _, t := range tabs {
		if t == notePath {
			return true
		}
	}
	return false
}

type restAPIConfig struct {
	APIKey string
	Port   int
}

func loadRESTAPIConfig(root string) *restAPIConfig {
	dataPath := filepath.Join(root, ".obsidian", "plugins", "obsidian-local-rest-api", "data.json")
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return nil
	}
	var raw struct {
		APIKey       string `json:"apiKey"`
		InsecurePort int    `json:"insecurePort"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.APIKey == "" {
		return nil
	}
	port := raw.InsecurePort
	if port == 0 {
		port = 27123
	}
	return &restAPIConfig{APIKey: raw.APIKey, Port: port}
}

func newRESTClient() *http.Client {
	return &http.Client{
		Timeout: 3 * time.Second,
	}
}

// isRESTAPIAvailable checks whether the Obsidian REST API is reachable.
func isRESTAPIAvailable(cfg *restAPIConfig) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", cfg.Port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// closeObsidianTab closes a tab by making it active then executing workspace:close.
func closeObsidianTab(cfg *restAPIConfig, relPath string) {
	client := newRESTClient()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)

	// Open the file to make it active
	openURL := fmt.Sprintf("%s/open/%s", baseURL, encodeRelPath(relPath))
	req, err := http.NewRequest("PUT", openURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Close the active tab
	closeURL := fmt.Sprintf("%s/commands/workspace%%3Aclose", baseURL)
	req, err = http.NewRequest("POST", closeURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// prepareCloseOldTabs checks REST API availability and workspace state,
// returning the config and filtered list of tabs to actually close.
// Warnings are printed synchronously so they're visible in Claude Code.
// Returns nil config if closing should be skipped.
func prepareCloseOldTabs(root string, paths []string) (*restAPIConfig, []string) {
	cfg := loadRESTAPIConfig(root)
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "[claude-code-hooks] Obsidian Local REST API プラグインが見つかりません。古いタブの自動クローズには同プラグインが必要です。\n")
		return nil, nil
	}

	if !isRESTAPIAvailable(cfg) {
		fmt.Fprintf(os.Stderr, "[claude-code-hooks] Obsidian REST API (http://127.0.0.1:%d) に接続できません。Obsidian が起動していることを確認してください。古いタブの自動クローズをスキップします。\n", cfg.Port)
		return nil, nil
	}

	// Check which files are actually open in Obsidian
	openFiles := openFilesInWorkspace(root)

	var toClose []string
	for _, p := range paths {
		relPath, err := filepath.Rel(root, p)
		if err != nil {
			continue
		}
		// Skip if not actually open (user may have closed it manually)
		if openFiles != nil && !openFiles[relPath] {
			continue
		}
		toClose = append(toClose, relPath)
	}
	return cfg, toClose
}

// closeTabsAsync closes the given tabs via REST API. Intended to run in a goroutine.
func closeTabsAsync(cfg *restAPIConfig, relPaths []string) {
	for _, relPath := range relPaths {
		closeObsidianTab(cfg, relPath)
	}
}

// encodeRelPath encodes each segment of a relative path for use in URLs.
func encodeRelPath(relPath string) string {
	parts := strings.Split(relPath, string(filepath.Separator))
	for i, p := range parts {
		// Manually percent-encode, preserving readability
		parts[i] = strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(p, "%", "%25"),
				" ", "%20"),
			"#", "%23")
	}
	return strings.Join(parts, "/")
}
