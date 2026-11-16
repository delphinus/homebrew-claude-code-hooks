package notify

import "testing"

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
