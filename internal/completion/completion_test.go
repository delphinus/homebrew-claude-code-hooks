package completion

import (
	"strings"
	"testing"
)

func TestScript(t *testing.T) {
	tests := []struct {
		shell   string
		wantErr bool
		contain string
	}{
		{"bash", false, "complete -F _claude_code_hooks claude-code-hooks"},
		{"zsh", false, "#compdef claude-code-hooks"},
		{"fish", false, "complete -c claude-code-hooks"},
		{"powershell", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got, err := Script(tt.shell)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Script(%q) error = %v, wantErr %v", tt.shell, err, tt.wantErr)
			}
			if !tt.wantErr && !strings.Contains(got, tt.contain) {
				t.Errorf("Script(%q) does not contain %q", tt.shell, tt.contain)
			}
		})
	}
}
