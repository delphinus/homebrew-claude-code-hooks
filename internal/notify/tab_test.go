package notify

import (
	"strings"
	"testing"
)

func TestTabLabelFrom(t *testing.T) {
	panes := []paneEntry{
		{WindowID: 0, TabID: 0, PaneID: 0, Title: "nvim ~"},
		{WindowID: 0, TabID: 1, PaneID: 1, Title: "⠂ FUJIYAMA-35661 の対応を進める"},
		{WindowID: 0, TabID: 1, PaneID: 7, Title: "zsh"},
		{WindowID: 0, TabID: 4, PaneID: 17, Title: "⠂ 通知をリッチにする", TabTitle: "claude"},
		{WindowID: 1, TabID: 9, PaneID: 20, Title: "別ウィンドウ"},
	}

	tests := []struct {
		name string
		pane string
		want string
	}{
		{"最初のタブ", "0", "1: nvim ~"},
		{"複数ペインのタブはペインタイトルを連結", "1", "2: ⠂ FUJIYAMA-35661 の対応を進める | zsh"},
		{"同じタブの別ペインでも同じラベル", "7", "2: ⠂ FUJIYAMA-35661 の対応を進める | zsh"},
		{"明示的な tab_title が優先される", "17", "3: claude"},
		{"タブ番号はウィンドウごとに数える", "20", "1: 別ウィンドウ"},
		{"未知のペインは空", "999", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tabLabelFrom(panes, tt.pane); got != tt.want {
				t.Errorf("tabLabelFrom(%q) = %q, want %q", tt.pane, got, tt.want)
			}
		})
	}
}

func TestTabLabelFrom_LongTitleIsTruncated(t *testing.T) {
	long := strings.Repeat("あ", 60)
	panes := []paneEntry{{WindowID: 0, TabID: 0, PaneID: 0, Title: long}}

	got := tabLabelFrom(panes, "0")
	want := "1: " + strings.Repeat("あ", maxTitleRunes-1) + "…"
	if got != want {
		t.Errorf("tabLabelFrom = %q, want %q", got, want)
	}
	if n := len([]rune(strings.TrimPrefix(got, "1: "))); n != maxTitleRunes {
		t.Errorf("truncated title has %d runes, want %d", n, maxTitleRunes)
	}
}

func TestTabLabelFrom_EmptyTitle(t *testing.T) {
	panes := []paneEntry{{WindowID: 0, TabID: 0, PaneID: 0, Title: ""}}
	if got, want := tabLabelFrom(panes, "0"), "1: "+untitled; got != want {
		t.Errorf("tabLabelFrom = %q, want %q", got, want)
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"", 5, ""},
		{"abcde", 5, "abcde"},
		{"abcdef", 5, "abcd…"},
		{"日本語のタイトル", 4, "日本語…"},
	}
	for _, tt := range tests {
		if got := truncateRunes(tt.in, tt.max); got != tt.want {
			t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
	}
}
