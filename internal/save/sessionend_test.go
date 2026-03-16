package save

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInsertSummary_New(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "test.md")

	content := "---\nid: test\n---\n# Hello\nsome content\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	insertSummary(notePath, "This is a summary.\nSecond line.")

	got, _ := os.ReadFile(notePath)
	want := "---\nid: test\n---\n\n> [!summary]\n> This is a summary.\n> Second line.\n\n# Hello\nsome content\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInsertSummary_Replace(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "test.md")

	content := "---\nid: test\n---\n\n> [!summary]\n> Old summary line1.\n> Old summary line2.\n\n# Hello\nsome content\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	insertSummary(notePath, "New summary.\nNew second line.")

	got, _ := os.ReadFile(notePath)
	want := "---\nid: test\n---\n\n> [!summary]\n> New summary.\n> New second line.\n\n# Hello\nsome content\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInsertSummary_ReplaceNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "test.md")

	// Summary at end of file with no trailing content
	content := "---\nid: test\n---\n\n> [!summary]\n> Old summary.\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	insertSummary(notePath, "Updated summary.")

	got, _ := os.ReadFile(notePath)
	want := "---\nid: test\n---\n\n> [!summary]\n> Updated summary.\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
