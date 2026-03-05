package save

import "testing"

func TestEnsureTableBlankLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "blank line before table",
			in:   "text\n| a | b |\n|---|---|\n| 1 | 2 |",
			want: "text\n\n| a | b |\n|---|---|\n| 1 | 2 |",
		},
		{
			name: "blank line after table",
			in:   "| a | b |\n|---|---|\n| 1 | 2 |\ntext",
			want: "| a | b |\n|---|---|\n| 1 | 2 |\n\ntext",
		},
		{
			name: "blank lines both sides",
			in:   "before\n| a | b |\n|---|---|\n| 1 | 2 |\nafter",
			want: "before\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nafter",
		},
		{
			name: "already has blank lines",
			in:   "before\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nafter",
			want: "before\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nafter",
		},
		{
			name: "table at start of text",
			in:   "| a | b |\n|---|---|\n| 1 | 2 |\nafter",
			want: "| a | b |\n|---|---|\n| 1 | 2 |\n\nafter",
		},
		{
			name: "table at end of text",
			in:   "before\n| a | b |\n|---|---|\n| 1 | 2 |",
			want: "before\n\n| a | b |\n|---|---|\n| 1 | 2 |",
		},
		{
			name: "no table",
			in:   "just text\nno tables here",
			want: "just text\nno tables here",
		},
		{
			name: "pipe inside code fence ignored",
			in:   "```\n| not a table |\n```",
			want: "```\n| not a table |\n```",
		},
		{
			name: "table after code fence",
			in:   "```\ncode\n```\n| a | b |\n|---|---|\n| 1 | 2 |",
			want: "```\ncode\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |",
		},
		{
			name: "real world case",
			in:   "There are 5 types:\n| Type | Icon |\n|---|---|\n| NOTE | info |\n\nUse Nerd Font for consistency.",
			want: "There are 5 types:\n\n| Type | Icon |\n|---|---|\n| NOTE | info |\n\nUse Nerd Font for consistency.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureTableBlankLines(tt.in)
			if got != tt.want {
				t.Errorf("EnsureTableBlankLines():\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
