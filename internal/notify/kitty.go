package notify

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/kitty"
)

// kittyTab resolves the subtitle label and the "is this pane being looked at
// right now?" flag for a kitty window, in a single `kitten @ ls` round trip.
//
// This is considerably less work than the WezTerm side needs: kitty reports the
// tab title directly (no reconstructing it from pane titles) and its OS windows
// carry is_focused, so no separate frontmost-process lookup is required.
//
// On any failure it returns ("", false): no subtitle, and the notification is
// shown rather than suppressed.
func kittyTab(window string) (label string, focused bool) {
	id, err := strconv.Atoi(window)
	if err != nil {
		return "", false
	}
	osWindows, err := kitty.List()
	if err != nil {
		return "", false
	}
	tab, index, focused, ok := kitty.Locate(osWindows, id)
	if !ok {
		return "", false
	}

	title := truncateRunes(strings.TrimSpace(tab.Title), maxTitleRunes)
	if title == "" {
		title = untitled
	}
	return fmt.Sprintf("%d: %s", index, title), focused
}
