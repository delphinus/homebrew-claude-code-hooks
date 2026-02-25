package note

import "testing"

func TestMakeDisplayTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello world", "hello world"},
		{"with invalid chars", `hello/world\test:file*name?"<>|`, "helloworldtestfilename"},
		{"long prompt", "a very long prompt that exceeds fifty characters and should be truncated here", "a very long prompt that exceeds fifty characters a"},
		{"multiline", "first line\nsecond line", "first line"},
		{"leading spaces", "   hello", "hello"},
		{"trailing spaces", "hello   ", "hello"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeDisplayTitle(tt.input)
			if got != tt.want {
				t.Errorf("MakeDisplayTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMakeFilenameTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello world", "hello-world"},
		{"multiple spaces", "hello   world", "hello-world"},
		{"with invalid chars", `hello/world`, "helloworld"},
		{"multiline", "first line\nsecond line", "first-line"},
		{"leading hyphens", "---hello", "hello"},
		{"trailing hyphens", "hello---", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeFilenameTitle(tt.input)
			if got != tt.want {
				t.Errorf("MakeFilenameTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMakeIDSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello-world", "hello-world"},
		{"uppercase", "Hello-World", "hello-world"},
		{"non-ascii removed", "こんにちは", ""}, // random fallback, just check non-empty
		{"mixed", "hello-世界-world", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeIDSlug(tt.input)
			if tt.want == "" {
				// Should be 4 random uppercase chars
				if len(got) != 4 {
					t.Errorf("MakeIDSlug(%q) = %q, expected 4-char random string", tt.input, got)
				}
			} else if got != tt.want {
				t.Errorf("MakeIDSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
