package save

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestNote creates a note file inside the temp vault and returns its path.
func newTestNote(t *testing.T, vaultDir, content string) string {
	t.Helper()
	path := filepath.Join(vaultDir, "note.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test note: %v", err)
	}
	return path
}

// makeUnwritable makes the note reject O_WRONLY opens, standing in for the
// vault-wide EPERM that iCloud has been observed returning.
func makeUnwritable(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
}

func TestAppendToFile_WritesDirectlyWhenVaultIsHealthy(t *testing.T) {
	vaultDir, _ := setupTestDirs(t)
	note := newTestNote(t, vaultDir, "head\n")

	if err := appendToFile(note, "body\n"); err != nil {
		t.Fatalf("appendToFile: %v", err)
	}

	got := readFile(t, note)
	if got != "head\nbody\n" {
		t.Errorf("note = %q, want %q", got, "head\nbody\n")
	}
	if s := readSpool(note); s != "" {
		t.Errorf("spool should be empty, got %q", s)
	}
}

func TestAppendToFile_SpoolsInsteadOfDroppingWhenWriteFails(t *testing.T) {
	vaultDir, _ := setupTestDirs(t)
	note := newTestNote(t, vaultDir, "head\n")
	makeUnwritable(t, note)

	// Spooling counts as success: the caller must record the content as
	// written, otherwise it is queued again on the next invocation.
	if err := appendToFile(note, "lost?\n"); err != nil {
		t.Fatalf("appendToFile should not report an error once spooled: %v", err)
	}

	if got := readFile(t, note); got != "head\n" {
		t.Errorf("note should be untouched, got %q", got)
	}
	if s := readSpool(note); s != "lost?\n" {
		t.Errorf("spool = %q, want %q", s, "lost?\n")
	}
}

func TestAppendToFile_FlushesSpoolInOrderOnceWritable(t *testing.T) {
	vaultDir, _ := setupTestDirs(t)
	note := newTestNote(t, vaultDir, "head\n")

	if err := os.Chmod(note, 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	for _, s := range []string{"first\n", "second\n"} {
		if err := appendToFile(note, s); err != nil {
			t.Fatalf("appendToFile while unwritable: %v", err)
		}
	}
	if err := os.Chmod(note, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := appendToFile(note, "third\n"); err != nil {
		t.Fatalf("appendToFile after recovery: %v", err)
	}

	want := "head\nfirst\nsecond\nthird\n"
	if got := readFile(t, note); got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
	if s := readSpool(note); s != "" {
		t.Errorf("spool should be cleared, got %q", s)
	}
}

func TestFlushPendingAppends_LandsSpoolWithoutNewContent(t *testing.T) {
	vaultDir, _ := setupTestDirs(t)
	note := newTestNote(t, vaultDir, "head\n")

	if err := writeSpool(note, "pending\n"); err != nil {
		t.Fatalf("writeSpool: %v", err)
	}
	flushPendingAppends(note)

	if got := readFile(t, note); got != "head\npending\n" {
		t.Errorf("note = %q, want %q", got, "head\npending\n")
	}
	if s := readSpool(note); s != "" {
		t.Errorf("spool should be cleared, got %q", s)
	}
}

func TestSpoolPath_DiffersPerNote(t *testing.T) {
	setupTestDirs(t)
	if spoolPath("/vault/a.md") == spoolPath("/vault/b.md") {
		t.Error("different notes must not share a spool file")
	}
}

func TestTruncateSpool_DropsOldestOnNewlineBoundary(t *testing.T) {
	// Well over the cap, so truncation kicks in. Multi-byte runes ensure a
	// naive byte cut would corrupt the result.
	line := strings.Repeat("あ", 100) + "\n"
	content := strings.Repeat(line, (maxSpoolBytes/len(line))+10)

	got := truncateSpool(content)

	if !strings.HasPrefix(got, spoolTruncatedNotice) {
		t.Error("truncated spool should carry the notice")
	}
	if len(got) > maxSpoolBytes+len(spoolTruncatedNotice) {
		t.Errorf("truncated spool is %d bytes, want <= %d", len(got), maxSpoolBytes+len(spoolTruncatedNotice))
	}
	if !strings.HasSuffix(got, line) {
		t.Error("newest content should be kept")
	}
	body := strings.TrimPrefix(got, spoolTruncatedNotice)
	if !strings.HasPrefix(body, "あ") {
		t.Errorf("cut should land on a line boundary, got prefix %q", body[:10])
	}
}

func TestTruncateSpool_LeavesSmallContentAlone(t *testing.T) {
	if got := truncateSpool("short\n"); got != "short\n" {
		t.Errorf("truncateSpool = %q, want %q", got, "short\n")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
