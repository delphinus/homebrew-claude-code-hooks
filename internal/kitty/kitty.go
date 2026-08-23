// Package kitty wraps the bits of kitty's remote control that more than one hook
// needs: locating the kitten binary and reading the window tree.
package kitty

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

// Window is one kitty window (what WezTerm calls a pane).
type Window struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	IsFocused bool   `json:"is_focused"`
}

// Tab is one tab, holding windows.
type Tab struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	IsFocused bool     `json:"is_focused"`
	Windows   []Window `json:"windows"`
}

// OSWindow is one OS-level window. IsFocused is true only when kitty itself is
// the frontmost application, which is what makes an explicit frontmost-process
// check (the osascript call the WezTerm code needed) unnecessary.
type OSWindow struct {
	ID        int   `json:"id"`
	IsFocused bool  `json:"is_focused"`
	Tabs      []Tab `json:"tabs"`
}

// Path locates the kitten binary. A hook may be launched with a minimal PATH, so
// fall back to the app bundle before giving up. Returns "" when not found.
func Path() string {
	if p, err := exec.LookPath("kitten"); err == nil {
		return p
	}
	for _, p := range []string{
		"/Applications/kitty.app/Contents/MacOS/kitten",
		filepath.Join(os.Getenv("HOME"), "Applications/kitty.app/Contents/MacOS/kitten"),
	} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// List runs `kitten @ ls` and returns the window tree. The instance to talk to
// comes from KITTY_LISTEN_ON, which hooks inherit from the kitty window they run in.
func List() ([]OSWindow, error) {
	kitten := Path()
	if kitten == "" {
		return nil, exec.ErrNotFound
	}
	out, err := exec.Command(kitten, "@", "ls").Output()
	if err != nil {
		return nil, err
	}
	var osWindows []OSWindow
	if err := json.Unmarshal(out, &osWindows); err != nil {
		return nil, err
	}
	return osWindows, nil
}

// Locate finds the given window id in the tree.
//
// It returns the tab holding it, the tab's 1-based position in its OS window
// (kitty numbers tabs by position in the tab bar, like the goto_tab shortcuts do)
// and whether that window is the one actually being looked at right now.
func Locate(osWindows []OSWindow, windowID int) (tab Tab, index int, focused bool, ok bool) {
	for _, osWindow := range osWindows {
		for i, t := range osWindow.Tabs {
			for _, w := range t.Windows {
				if w.ID != windowID {
					continue
				}
				return t, i + 1, osWindow.IsFocused && w.IsFocused, true
			}
		}
	}
	return Tab{}, 0, false, false
}
