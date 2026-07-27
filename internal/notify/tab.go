package notify

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// maxTitleRunes はサブタイトルに載せるタブタイトルの最大文字数。
// これを超えるタイトルは末尾を省略記号に置き換える。
const maxTitleRunes = 40

// untitled はタブタイトルが空のときの表示。
const untitled = "(無題)"

// paneEntry is a subset of `wezterm cli list --format json` の 1 エントリ。
type paneEntry struct {
	WindowID int    `json:"window_id"`
	TabID    int    `json:"tab_id"`
	PaneID   int    `json:"pane_id"`
	Title    string `json:"title"`
	TabTitle string `json:"tab_title"`
}

// tabLabel returns a "<タブ番号>: <タブタイトル>" label for the given WezTerm pane,
// matching what the WezTerm tab bar shows. Returns "" when it cannot be resolved
// (wezterm CLI unavailable, pane not found, ...) so the caller can simply omit it.
func tabLabel(pane string) string {
	out, err := exec.Command("wezterm", "cli", "list", "--format", "json").Output()
	if err != nil {
		return ""
	}
	var panes []paneEntry
	if err := json.Unmarshal(out, &panes); err != nil {
		return ""
	}
	return tabLabelFrom(panes, pane)
}

// tabLabelFrom is the pure part of tabLabel, split out for testing.
func tabLabelFrom(panes []paneEntry, pane string) string {
	cur, ok := findPane(panes, pane)
	if !ok {
		return ""
	}

	title := truncateRunes(strings.TrimSpace(tabTitle(panes, cur)), maxTitleRunes)
	if title == "" {
		title = untitled
	}
	return fmt.Sprintf("%d: %s", tabIndex(panes, cur)+1, title)
}

func findPane(panes []paneEntry, pane string) (paneEntry, bool) {
	for _, p := range panes {
		if fmt.Sprintf("%d", p.PaneID) == pane {
			return p, true
		}
	}
	return paneEntry{}, false
}

// tabIndex returns the 0-based position of cur's tab within its window.
// WezTerm exposes no tab index over the CLI, so it is derived from the order of
// `wezterm cli list`, which walks each window's tabs in tab-bar order.
func tabIndex(panes []paneEntry, cur paneEntry) int {
	index := 0
	seen := map[int]bool{}
	for _, p := range panes {
		if p.WindowID != cur.WindowID || seen[p.TabID] {
			continue
		}
		seen[p.TabID] = true
		if p.TabID == cur.TabID {
			return index
		}
		index++
	}
	return index
}

// tabTitle mirrors the tab title logic of the WezTerm side (format-tab-title):
// an explicit tab title wins, otherwise the pane titles of the tab are joined.
func tabTitle(panes []paneEntry, cur paneEntry) string {
	if cur.TabTitle != "" {
		return cur.TabTitle
	}

	var titles []string
	seen := map[string]bool{}
	for _, p := range panes {
		if p.WindowID != cur.WindowID || p.TabID != cur.TabID {
			continue
		}
		if p.Title == "" || seen[p.Title] {
			continue
		}
		seen[p.Title] = true
		titles = append(titles, p.Title)
	}
	return strings.Join(titles, " | ")
}

// truncateRunes shortens s to at most max runes, marking the cut with "…".
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
