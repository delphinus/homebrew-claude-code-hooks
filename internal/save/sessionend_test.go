package save

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
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

func TestLastActivityTime(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\nid: x\ndate: 2026-06-29T11:28:00\n---\n\n## Assistant (11:30:00)\n\nhi\n\n## Assistant (18:02:33)\n\nbye\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-06-29T11:28:00"); got != "2026-06-29T18:02:33" {
		t.Errorf("got %q, want %q", got, "2026-06-29T18:02:33")
	}
}

// The headings actually written by handleStop / recordLastAssistantMessage
// carry the 🤖 emoji, so that is the form that has to match.
func TestLastActivityTime_EmojiHeading(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\nid: x\ndate: 2026-06-29T11:28:00\n---\n\n## 🤖 Assistant (11:30:00)\n\nhi\n\n## 🤖 Assistant (18:02:33)\n\nbye\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-06-29T11:28:00"); got != "2026-06-29T18:02:33" {
		t.Errorf("got %q, want %q", got, "2026-06-29T18:02:33")
	}
}

func TestLastActivityTime_CrossMidnight(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n## Assistant (23:50:00)\n\n## Assistant (00:30:00)\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-06-25T23:40:00"); got != "2026-06-26T00:30:00" {
		t.Errorf("got %q, want %q (day should roll over)", got, "2026-06-26T00:30:00")
	}
}

func TestLastActivityTime_NoHeading(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	os.WriteFile(notePath, []byte("---\n---\n\nno timestamped entries (12:00:00)\n"), 0o644)

	if got := lastActivityTime(notePath, "2026-06-29T11:28:00"); got != "" {
		t.Errorf("expected empty for no heading, got %q", got)
	}
}

// A note split off by a cwd change can hold callouts only (no Assistant
// heading). Its end must come from the callouts, not from the quit time.
func TestLastActivityTime_CalloutsOnly(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n> [!question] User (11:25:40)\n> do it\n\n" +
		"> [!terminal]- 作業ディレクトリを作る (11:25:32)\n> ```bash\n> mkdir x\n> ```\n\n" +
		"> [!file] Edit: a.go (11:31:05)\n\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-11-08T11:25:31"); got != "2026-11-08T11:31:05" {
		t.Errorf("got %q, want %q", got, "2026-11-08T11:31:05")
	}
}

// Callouts after the last Assistant heading (e.g. a tool run that was
// interrupted) are activity too.
func TestLastActivityTime_CalloutAfterHeading(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n## 🤖 Assistant (10:00:00)\n\n> [!terminal]- run (10:20:00)\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-11-08T09:59:00"); got != "2026-11-08T10:20:00" {
		t.Errorf("got %q, want %q", got, "2026-11-08T10:20:00")
	}
}

// Walking the entries in order, each backward jump is another day, so a note
// that stays open over two midnights ends on the right date.
func TestLastActivityTime_TwoMidnights(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n## Assistant (23:00:00)\n\n## Assistant (12:00:00)\n\n## Assistant (09:00:00)\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-11-06T22:21:00"); got != "2026-11-08T09:00:00" {
		t.Errorf("got %q, want %q", got, "2026-11-08T09:00:00")
	}
}

// A spooled append can land a little out of order; that is not a new day.
func TestLastActivityTime_SlightlyOutOfOrder(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n## Assistant (10:05:00)\n\n> [!file] Write: a (10:04:58)\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-11-08T10:00:00"); got != "2026-11-08T10:05:00" {
		t.Errorf("got %q, want %q", got, "2026-11-08T10:05:00")
	}
}

// Parenthesised times in the body (not entry headers) are not activity.
func TestLastActivityTime_IgnoresBodyTimes(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\n---\n\n## Assistant (10:00:00)\n\nmeeting starts (16:30:00)\n> quoted (17:00:00)\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	if got := lastActivityTime(notePath, "2026-11-08T09:00:00"); got != "2026-11-08T10:00:00" {
		t.Errorf("got %q, want %q", got, "2026-11-08T10:00:00")
	}
}

func TestSetEnded(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "n.md")
	content := "---\nid: x\naliases:\n  - T\ntags:\n  - claude-code\ndate: 2026-06-29T11:28:00\nsession_id: s\nhostname: h\ncwd: /tmp\n---\n\nbody\n"
	os.WriteFile(notePath, []byte(content), 0o644)

	setEnded(notePath, "2026-06-29T18:02:33")

	got, _ := os.ReadFile(notePath)
	if !strings.Contains(string(got), "date: 2026-06-29T11:28:00\nended: 2026-06-29T18:02:33\n") {
		t.Errorf("ended not placed right after date:\n%s", got)
	}
	fm, _, err := frontmatter.Parse(string(got))
	if err != nil || fm.Ended != "2026-06-29T18:02:33" {
		t.Errorf("parse back ended = %q, err = %v", fm.Ended, err)
	}
}

func TestSummaryTimeLine(t *testing.T) {
	if got := summaryTimeLine("2026-06-29T10:36:00", "2026-06-29T18:02:00"); got != "⏱ 10:36–18:02 (7h26m)" {
		t.Errorf("got %q", got)
	}
	if got := summaryTimeLine("2026-06-25T23:40:00", "2026-06-26T00:30:00"); got != "⏱ 23:40–00:30(+1d) (0h50m)" {
		t.Errorf("cross-day got %q", got)
	}
	if got := summaryTimeLine("", "x"); got != "" {
		t.Errorf("expected empty for unparseable, got %q", got)
	}
}
