package filter

import "testing"

func TestShouldRecordCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ls", false},
		{"ls -la", false},
		{"cat foo.txt", false},
		{"grep pattern file | sort | uniq", false},
		{"go build ./...", true},
		{"npm install", true},
		{"ls | go build", true},
		{"cat foo && npm test", true},
		{"echo hello; python script.py", true},
		{"", false},
		{"git status", true},
		{"docker compose up", true},
		{"mkdir -p /tmp/test && cd /tmp/test", false},
		{"rm -rf dist && go build", true},
		{"jq .foo bar.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := ShouldRecordCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("ShouldRecordCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
