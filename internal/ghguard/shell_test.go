package ghguard

import (
	"reflect"
	"testing"
)

func TestSegments(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    [][]string
	}{
		{
			name:    "plain command",
			command: "gh pr view 5",
			want:    [][]string{{"gh", "pr", "view", "5"}},
		},
		{
			name:    "and-and",
			command: "echo hi && gh pr merge 5",
			want:    [][]string{{"echo", "hi"}, {"gh", "pr", "merge", "5"}},
		},
		{
			name:    "pipeline and semicolon",
			command: "cat f | gh pr comment 5; echo done",
			want:    [][]string{{"cat", "f"}, {"gh", "pr", "comment", "5"}, {"echo", "done"}},
		},
		{
			name:    "double quotes keep separators inside a word",
			command: `gh issue comment 5 --body "a; b && c"`,
			want:    [][]string{{"gh", "issue", "comment", "5", "--body", "a; b && c"}},
		},
		{
			name:    "single quotes are literal",
			command: `gh api -f query='mutation{x}'`,
			want:    [][]string{{"gh", "api", "-f", "query=mutation{x}"}},
		},
		{
			name:    "command substitution starts a segment",
			command: `echo "$(gh release create v1)"`,
			want:    [][]string{{"echo"}, {"gh", "release", "create", "v1"}},
		},
		{
			name:    "backslash escapes",
			command: `gh pr comment 5 --body a\ b`,
			want:    [][]string{{"gh", "pr", "comment", "5", "--body", "a b"}},
		},
		{
			name:    "redirection ends the segment",
			command: "gh pr view 5 > out.txt",
			want:    [][]string{{"gh", "pr", "view", "5"}, {"out.txt"}},
		},
		{
			name:    "newlines separate",
			command: "gh pr view 5\ngh pr merge 5",
			want:    [][]string{{"gh", "pr", "view", "5"}, {"gh", "pr", "merge", "5"}},
		},
		{
			name:    "empty",
			command: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments(tt.command)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("segments(%q) = %#v, want %#v", tt.command, got, tt.want)
			}
		})
	}
}
