package ghguard

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
)

// isolate clears the ambient environment so tests do not depend on the machine
// they run on.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv(hostsEnv, "")
	t.Setenv("GH_HOST", "")
}

func TestFindWrite(t *testing.T) {
	isolate(t)

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"read: pr view", "gh pr view 5", false},
		{"read: pr checkout", "gh pr checkout 5", false},
		{"read: repo clone", "gh repo clone owner/repo", false},
		{"read: auth status", "gh auth status", false},
		{"read: secret list", "gh secret list", false},
		{"read: search", "gh search issues foo", false},
		{"read: not gh at all", "git push origin main", false},
		{"read: gh in a word", "highlight --gh", false},
		{"read: empty", "   ", false},

		{"write: pr comment", "gh pr comment 5 --body hi", true},
		{"write: issue create", "gh issue create --title x --body y", true},
		{"write: pr merge", "gh pr merge 5 --squash", true},
		{"write: release create", "gh release create v1.0.0", true},
		{"write: secret set", "gh secret set FOO --body bar", true},
		{"write: workflow run", "gh workflow run deploy.yml", true},
		{"write: unknown subcommand errs towards asking", "gh newthing do", true},

		{"api: plain GET reads", "gh api repos/o/r/issues", false},
		{"api: explicit GET reads", "gh api --method GET repos/o/r", false},
		{"api: POST writes", "gh api -X POST repos/o/r/issues", true},
		{"api: DELETE writes", "gh api --method DELETE repos/o/r", true},
		{"api: fields imply POST", "gh api repos/o/r/issues -f title=x", true},
		{"api: graphql mutation writes", "gh api graphql -f query='mutation{x}'", true},

		{"chain: after &&", "echo hi && gh pr merge 5", true},
		{"chain: after ;", "cd /tmp; gh issue close 3", true},
		{"chain: in a pipeline", "cat body.md | gh pr comment 5 --body-file -", true},
		{"chain: command substitution", `echo "$(gh release create v1)"`, true},
		{"chain: read then read", "gh pr view 5 && gh pr diff 5", false},

		{"quoting: write verb inside a quoted body", `gh issue comment 5 --body "done; gh pr merge 5"`, true},
		{"quoting: gh named only inside a quoted string", `echo "gh pr merge 5"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findWrite(tt.command, "") != nil
			if got != tt.want {
				t.Errorf("findWrite(%q) ask = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestFindWriteHostSelection(t *testing.T) {
	isolate(t)

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"default host is guarded", "gh pr comment 5 --body x", true},
		{"--repo with owner/repo keeps the default host", "gh pr comment --repo o/r 5 -b x", true},
		{"--repo with a URL", "gh issue comment --repo https://github.com/o/r 5 -b x", true},
		{"--repo pointing at another host", "gh pr comment --repo ghe.example.com/o/r 5 -b x", false},
		{"--repo URL on another host", "gh pr comment --repo https://ghe.example.com/o/r 5 -b x", false},
		{"--hostname flag", "gh api --hostname ghe.example.com -X POST repos/o/r/issues", false},
		{"GH_HOST assignment", "GH_HOST=ghe.example.com gh issue create --title x", false},
		{"GH_HOST via env wrapper", "env GH_HOST=ghe.example.com gh issue create --title x", false},
		{"host with a port", "gh pr comment --repo ghe.example.com:8080/o/r 5 -b x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findWrite(tt.command, "") != nil
			if got != tt.want {
				t.Errorf("findWrite(%q) ask = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestFindWriteProcessGHHost(t *testing.T) {
	isolate(t)
	t.Setenv("GH_HOST", "ghe.example.com")

	if findWrite("gh issue create --title x", "") != nil {
		t.Error("GH_HOST in the environment should move the write off the guarded host")
	}
}

func TestGuardedHostsOverride(t *testing.T) {
	isolate(t)
	t.Setenv(hostsEnv, "ghe.example.com, Example.COM ")

	if findWrite("gh pr comment 5 -b x", "") != nil {
		t.Error("github.com should not be guarded once the list is overridden")
	}
	if findWrite("gh pr comment --repo ghe.example.com/o/r 5 -b x", "") == nil {
		t.Error("a host from the override list should be guarded")
	}
	if findWrite("gh pr comment --repo example.com/o/r 5 -b x", "") == nil {
		t.Error("the override list should be matched case-insensitively")
	}
}

func TestFindWriteDetails(t *testing.T) {
	isolate(t)

	call := findWrite("gh pr comment --repo o/r 5 --body hi", "")
	if call == nil {
		t.Fatal("expected a write")
	}
	if call.host != "github.com" {
		t.Errorf("host = %q, want %q", call.host, "github.com")
	}
	if call.repo != "o/r" {
		t.Errorf("repo = %q, want %q", call.repo, "o/r")
	}
	if call.summary != "gh pr comment" {
		t.Errorf("summary = %q, want %q", call.summary, "gh pr comment")
	}
}

func TestEmit(t *testing.T) {
	var buf strings.Builder
	if err := emit(&buf, writeCall{host: "github.com", repo: "o/r", summary: "gh pr comment"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got decision
	if err := json.Unmarshal([]byte(buf.String()), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	out := got.HookSpecificOutput
	if out.HookEventName != "PreToolUse" {
		t.Errorf("hookEventName = %q, want %q", out.HookEventName, "PreToolUse")
	}
	if out.PermissionDecision != "ask" {
		t.Errorf("permissionDecision = %q, want %q", out.PermissionDecision, "ask")
	}
	for _, want := range []string{"github.com", "o/r", "gh pr comment"} {
		if !strings.Contains(out.PermissionDecisionReason, want) {
			t.Errorf("reason %q does not mention %q", out.PermissionDecisionReason, want)
		}
	}
}

func TestRunIgnoresOtherTools(t *testing.T) {
	isolate(t)

	if err := Run(nil); err != nil {
		t.Errorf("Run(nil) = %v, want nil", err)
	}
	input := &hookdata.HookInput{ToolName: "Edit"}
	input.ToolInput.Command = "gh pr merge 5"
	if err := Run(input); err != nil {
		t.Errorf("Run(Edit) = %v, want nil", err)
	}
}

func TestHostFromGitURL(t *testing.T) {
	tests := map[string]string{
		"git@github.com:owner/repo.git":          "github.com",
		"ssh://git@ghe.example.com/owner/repo":   "ghe.example.com",
		"https://github.com/owner/repo.git":      "github.com",
		"https://ghe.example.com:8443/owner/rep": "ghe.example.com",
		"":                                       "",
		"owner/repo":                             "",
	}
	for remote, want := range tests {
		if got := hostFromGitURL(remote); got != want {
			t.Errorf("hostFromGitURL(%q) = %q, want %q", remote, got, want)
		}
	}
}

func TestRepoName(t *testing.T) {
	tests := map[string]string{
		"o/r":                        "o/r",
		"ghe.example.com/o/r":        "o/r",
		"https://github.com/o/r":     "o/r",
		"https://github.com/o/r.git": "o/r",
		"":                           "",
	}
	for repo, want := range tests {
		if got := repoName(repo); got != want {
			t.Errorf("repoName(%q) = %q, want %q", repo, got, want)
		}
	}
}

func TestEnvAssignment(t *testing.T) {
	if name, value, ok := envAssignment("GH_HOST=x"); !ok || name != "GH_HOST" || value != "x" {
		t.Errorf("envAssignment(GH_HOST=x) = %q, %q, %v", name, value, ok)
	}
	if _, _, ok := envAssignment("=x"); ok {
		t.Error("a leading = is not an assignment")
	}
	if _, _, ok := envAssignment("./foo=bar"); ok {
		t.Error("a path is not an assignment")
	}
	if _, _, ok := envAssignment("gh"); ok {
		t.Error("a bare word is not an assignment")
	}
}
