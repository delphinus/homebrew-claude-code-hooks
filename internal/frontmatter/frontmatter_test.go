package frontmatter

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	content := `---
id: 20250101-120000-abc1-test
aliases:
  - Test Title
tags:
  - claude-code
date: 2025-01-01T12:00:00
session_id: abc123
hostname: myhost
cwd: /tmp/project
claude_version: 2.1.74 (Claude Code)
related:
  - "[[other-note]]"
---

## User (12:00:00)

Hello
`

	fm, body, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.ID != "20250101-120000-abc1-test" {
		t.Errorf("ID = %q, want %q", fm.ID, "20250101-120000-abc1-test")
	}
	if len(fm.Aliases) != 1 || fm.Aliases[0] != "Test Title" {
		t.Errorf("Aliases = %v, want [Test Title]", fm.Aliases)
	}
	if len(fm.Tags) != 1 || fm.Tags[0] != "claude-code" {
		t.Errorf("Tags = %v, want [claude-code]", fm.Tags)
	}
	if fm.Date != "2025-01-01T12:00:00" {
		t.Errorf("Date = %q, want %q", fm.Date, "2025-01-01T12:00:00")
	}
	if fm.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", fm.SessionID, "abc123")
	}
	if fm.Hostname != "myhost" {
		t.Errorf("Hostname = %q, want %q", fm.Hostname, "myhost")
	}
	if fm.CWD != "/tmp/project" {
		t.Errorf("CWD = %q, want %q", fm.CWD, "/tmp/project")
	}
	if fm.ClaudeVersion != "2.1.74 (Claude Code)" {
		t.Errorf("ClaudeVersion = %q, want %q", fm.ClaudeVersion, "2.1.74 (Claude Code)")
	}
	if len(fm.Related) != 1 || fm.Related[0] != `"[[other-note]]"` {
		t.Errorf("Related = %v, want [\"[[other-note]]\"]", fm.Related)
	}
	if !strings.Contains(body, "## User (12:00:00)") {
		t.Errorf("body should contain user section, got: %q", body)
	}
}

func TestParseNoRelated(t *testing.T) {
	content := `---
id: test-id
aliases:
  - Title
tags:
  - claude-code
date: 2025-01-01T12:00:00
session_id: abc123
hostname: myhost
cwd: /tmp
---

Body
`

	fm, _, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fm.Related) != 0 {
		t.Errorf("Related should be empty, got %v", fm.Related)
	}
}

func TestRender(t *testing.T) {
	fm := &Frontmatter{
		ID:        "20250101-120000-abc1-test",
		Aliases:   []string{"Test Title"},
		Tags:      []string{"claude-code"},
		Date:      "2025-01-01T12:00:00",
		SessionID: "abc123",
		Hostname:  "myhost",
		CWD:       "/tmp/project",
	}

	rendered := fm.Render()
	if !strings.HasPrefix(rendered, "---\n") {
		t.Error("should start with ---")
	}
	if !strings.HasSuffix(rendered, "---\n") {
		t.Error("should end with ---")
	}
	if strings.Contains(rendered, "related:") {
		t.Error("should not contain related when empty")
	}
}

func TestRenderWithRelated(t *testing.T) {
	fm := &Frontmatter{
		ID:        "test-id",
		Aliases:   []string{"Title"},
		Tags:      []string{"claude-code"},
		Date:      "2025-01-01T12:00:00",
		SessionID: "abc123",
		Hostname:  "myhost",
		CWD:       "/tmp",
		Related:   []string{`"[[other-note]]"`},
	}

	rendered := fm.Render()
	if !strings.Contains(rendered, "related:\n") {
		t.Error("should contain related section")
	}
	if !strings.Contains(rendered, `  - "[[other-note]]"`) {
		t.Error("should contain related link")
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	_, _, err := Parse("Just a regular file")
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestRoundTrip(t *testing.T) {
	fm := &Frontmatter{
		ID:            "20250101-120000-abc1-test",
		Aliases:       []string{"Test Title"},
		Tags:          []string{"claude-code"},
		Date:          "2025-01-01T12:00:00",
		SessionID:     "abc123",
		Hostname:      "myhost",
		CWD:           "/tmp/project",
		ClaudeVersion: "2.1.74 (Claude Code)",
		Related:       []string{`"[[other-note]]"`},
	}

	rendered := fm.Render()
	parsed, body, err := Parse(rendered + "\nBody text\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.ID != fm.ID {
		t.Errorf("ID mismatch: %q vs %q", parsed.ID, fm.ID)
	}
	if parsed.SessionID != fm.SessionID {
		t.Errorf("SessionID mismatch: %q vs %q", parsed.SessionID, fm.SessionID)
	}
	if parsed.ClaudeVersion != fm.ClaudeVersion {
		t.Errorf("ClaudeVersion mismatch: %q vs %q", parsed.ClaudeVersion, fm.ClaudeVersion)
	}
	if !strings.Contains(body, "Body text") {
		t.Errorf("body should contain 'Body text', got %q", body)
	}
}
