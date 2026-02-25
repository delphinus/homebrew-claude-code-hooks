package hookdata

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
}

type HookInput struct {
	SessionID            string    `json:"session_id"`
	HookEventName        string    `json:"hook_event_name"`
	CWD                  string    `json:"cwd"`
	Prompt               string    `json:"prompt"`
	LastAssistantMessage string    `json:"last_assistant_message"`
	ToolName             string    `json:"tool_name"`
	ToolInput            ToolInput `json:"tool_input"`
}

// ReadFromStdin reads and parses a HookInput JSON from stdin.
func ReadFromStdin() (*HookInput, error) {
	return ReadFrom(os.Stdin)
}

// ReadFrom reads and parses a HookInput JSON from the given reader.
func ReadFrom(r io.Reader) (*HookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return &input, nil
}
