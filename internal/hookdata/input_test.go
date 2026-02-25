package hookdata

import (
	"strings"
	"testing"
)

func TestReadFrom(t *testing.T) {
	input := `{
		"session_id": "abc123",
		"hook_event_name": "UserPromptSubmit",
		"cwd": "/tmp/project",
		"prompt": "hello world",
		"tool_name": "Bash",
		"tool_input": {
			"command": "go build ./...",
			"description": "Build the project"
		}
	}`

	hi, err := ReadFrom(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hi.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", hi.SessionID, "abc123")
	}
	if hi.HookEventName != "UserPromptSubmit" {
		t.Errorf("HookEventName = %q, want %q", hi.HookEventName, "UserPromptSubmit")
	}
	if hi.CWD != "/tmp/project" {
		t.Errorf("CWD = %q, want %q", hi.CWD, "/tmp/project")
	}
	if hi.Prompt != "hello world" {
		t.Errorf("Prompt = %q, want %q", hi.Prompt, "hello world")
	}
	if hi.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want %q", hi.ToolName, "Bash")
	}
	if hi.ToolInput.Command != "go build ./..." {
		t.Errorf("ToolInput.Command = %q, want %q", hi.ToolInput.Command, "go build ./...")
	}
	if hi.ToolInput.Description != "Build the project" {
		t.Errorf("ToolInput.Description = %q, want %q", hi.ToolInput.Description, "Build the project")
	}
}

func TestReadFromInvalidJSON(t *testing.T) {
	_, err := ReadFrom(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadFromEmptyFields(t *testing.T) {
	hi, err := ReadFrom(strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hi.SessionID != "" {
		t.Errorf("expected empty SessionID, got %q", hi.SessionID)
	}
}
