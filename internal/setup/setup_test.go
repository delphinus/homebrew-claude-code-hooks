package setup

import (
	"encoding/json"
	"testing"
)

func TestIsOwnCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"claude-code-hooks save", true},
		{"claude-code-hooks tabcolor idle", true},
		{"/opt/homebrew/bin/claude-code-hooks save", true},
		{"'claude-code-hooks' save", true},
		{`"claude-code-hooks" save`, true},
		{"'/Users/foo/AgentStatus/hooks/claude-event-hook.sh' Idle # notchbar-agents-claude-hook", false},
		{"some-other-tool run", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isOwnCommand(c.cmd); got != c.want {
			t.Errorf("isOwnCommand(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

// hooksFromJSON is a small helper to build the generic JSON shape used by settings.
func hooksFromJSON(t *testing.T, s string) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad JSON in test: %v", err)
	}
	return v
}

// collectCommands returns every hook command string under an event, in order.
func collectCommands(t *testing.T, merged map[string]interface{}, event string) []string {
	t.Helper()
	var out []string
	entries, _ := merged[event].([]interface{})
	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		hooksArr, _ := entry["hooks"].([]interface{})
		for _, h := range hooksArr {
			ho, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := ho["command"].(string); ok {
				out = append(out, cmd)
			}
		}
	}
	return out
}

func TestMergeHooksPreservesForeignAndReplacesOwn(t *testing.T) {
	// Existing settings: own hooks mixed with a foreign hook in the same entry
	// (Stop), a foreign-only standalone event (PreToolUse), and a stale own hook
	// that no longer exists in hooks.json (Stop "oldcmd").
	existing := hooksFromJSON(t, `{
		"Stop": [
			{"hooks": [
				{"type":"command","command":"claude-code-hooks oldcmd"},
				{"type":"command","command":"'/x/claude-event-hook.sh' Idle # notchbar-agents-claude-hook"}
			]}
		],
		"PreToolUse": [
			{"hooks": [
				{"type":"command","command":"'/x/claude-event-hook.sh' Working # notchbar-agents-claude-hook"}
			]}
		]
	}`)

	// Fresh own hooks from hooks.json.
	own := hooksFromJSON(t, `{
		"Stop": [
			{"hooks": [
				{"type":"command","command":"claude-code-hooks tabcolor idle"},
				{"type":"command","command":"claude-code-hooks save"}
			]}
		]
	}`)

	merged := mergeHooks(existing, own)

	// Foreign PreToolUse hook must survive untouched.
	pre := collectCommands(t, merged, "PreToolUse")
	if len(pre) != 1 || pre[0] != "'/x/claude-event-hook.sh' Working # notchbar-agents-claude-hook" {
		t.Errorf("PreToolUse foreign hook not preserved: %v", pre)
	}

	// Stop: stale own hook removed, fresh own hooks present, foreign preserved.
	stop := collectCommands(t, merged, "Stop")
	joined := map[string]bool{}
	for _, c := range stop {
		joined[c] = true
	}
	if joined["claude-code-hooks oldcmd"] {
		t.Errorf("stale own hook was not removed: %v", stop)
	}
	if !joined["claude-code-hooks tabcolor idle"] || !joined["claude-code-hooks save"] {
		t.Errorf("fresh own hooks missing: %v", stop)
	}
	if !joined["'/x/claude-event-hook.sh' Idle # notchbar-agents-claude-hook"] {
		t.Errorf("foreign hook in mixed entry was not preserved: %v", stop)
	}

	// Own hooks should be ordered before foreign hooks.
	if stop[len(stop)-1] != "'/x/claude-event-hook.sh' Idle # notchbar-agents-claude-hook" {
		t.Errorf("foreign hook should come last, got order: %v", stop)
	}
}

func TestMergeHooksNilExisting(t *testing.T) {
	own := hooksFromJSON(t, `{
		"SessionStart": [
			{"hooks": [{"type":"command","command":"claude-code-hooks tabcolor startup"}]}
		]
	}`)
	merged := mergeHooks(nil, own)
	got := collectCommands(t, merged, "SessionStart")
	if len(got) != 1 || got[0] != "claude-code-hooks tabcolor startup" {
		t.Errorf("nil-existing merge = %v", got)
	}
}

func TestMergeHooksDropsEmptyForeignOnlyOwnEvent(t *testing.T) {
	// An existing event that contained ONLY an own hook should not leave an
	// empty entry behind after stripping, before the fresh own hook is added.
	existing := hooksFromJSON(t, `{
		"SessionStart": [
			{"hooks": [{"type":"command","command":"claude-code-hooks tabcolor startup"}]}
		]
	}`)
	own := hooksFromJSON(t, `{
		"SessionStart": [
			{"hooks": [{"type":"command","command":"claude-code-hooks tabcolor startup"}]}
		]
	}`)
	merged := mergeHooks(existing, own)
	got := collectCommands(t, merged, "SessionStart")
	if len(got) != 1 {
		t.Errorf("expected exactly one own hook after re-merge, got %v", got)
	}
}
