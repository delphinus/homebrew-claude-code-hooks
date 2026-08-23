package notify

import (
	"reflect"
	"testing"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/kitty"
)

func TestHelperArgs(t *testing.T) {
	got := helperArgs("Claude Code", "3: nvim ~", "作業が完了しました", "10", "kitty")
	want := []string{
		"post",
		"--title", "Claude Code",
		"--subtitle", "3: nvim ~",
		"--message", "作業が完了しました",
		"--pane", "10",
		"--term", "kitty",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("helperArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestHelperArgs_EmptySubtitle(t *testing.T) {
	got := helperArgs("Claude Code", "", "作業が完了しました", "10", "wezterm")
	want := []string{
		"post",
		"--title", "Claude Code",
		"--message", "作業が完了しました",
		"--pane", "10",
		"--term", "wezterm",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("helperArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestKittyLocate(t *testing.T) {
	tree := []kitty.OSWindow{
		{
			ID:        1,
			IsFocused: false,
			Tabs: []kitty.Tab{
				{ID: 1, Title: "shell", Windows: []kitty.Window{{ID: 1, IsFocused: true}}},
			},
		},
		{
			ID:        2,
			IsFocused: true,
			Tabs: []kitty.Tab{
				{ID: 5, Title: "one", Windows: []kitty.Window{{ID: 10}}},
				{ID: 6, Title: "two", Windows: []kitty.Window{{ID: 20, IsFocused: true}, {ID: 21}}},
			},
		},
	}

	for _, tc := range []struct {
		name    string
		id      int
		index   int
		title   string
		focused bool
		ok      bool
	}{
		// タブ番号は OS ウィンドウ内の並び順 (goto_tab と同じ数え方)。
		{"1 番目のタブ", 10, 1, "one", false, true},
		{"2 番目のタブ、見られているペイン", 20, 2, "two", true, true},
		{"同じタブの別ペインは見られていない", 21, 2, "two", false, true},
		// OS ウィンドウが最前面でなければ、ペインが is_focused でも「見られていない」。
		{"背面の OS ウィンドウ", 1, 1, "shell", false, true},
		{"知らない id", 999, 0, "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tab, index, focused, ok := kitty.Locate(tree, tc.id)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if index != tc.index || tab.Title != tc.title || focused != tc.focused {
				t.Errorf("got (index=%d title=%q focused=%v), want (index=%d title=%q focused=%v)",
					index, tab.Title, focused, tc.index, tc.title, tc.focused)
			}
		})
	}
}
