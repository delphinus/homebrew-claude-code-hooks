package save

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
)

// Appends into the Obsidian vault can fail transiently: iCloud has been
// observed returning EPERM for every note in the vault for minutes at a time.
// A hook invocation only ever sees the *current* last assistant message, so an
// append that fails is never retried on its own and the content disappears
// from the note for good. Failed appends are spooled to the local cache dir
// (not the vault, which is the thing that just failed) and flushed on the next
// append that succeeds.

// maxSpoolBytes caps the spool so a vault that stays unwritable cannot grow it
// without bound. The oldest content is dropped first.
const maxSpoolBytes = 1 << 20

const spoolTruncatedNotice = "> [!warning] 追記の保留分が上限を超えたため、古い分を破棄しました\n\n"

// spoolPath returns the local spool file for notePath. The note path is hashed
// so the key stays a valid filename whatever the note is called.
func spoolPath(notePath string) string {
	sum := sha256.Sum256([]byte(notePath))
	return filepath.Join(config.CacheDir(), "pending", hex.EncodeToString(sum[:])+".md")
}

// readSpool returns the content queued for notePath, or "" if there is none.
func readSpool(notePath string) string {
	data, err := os.ReadFile(spoolPath(notePath))
	if err != nil {
		return ""
	}
	return string(data)
}

func writeSpool(notePath, content string) error {
	path := spoolPath(notePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(truncateSpool(content)), 0o644)
}

func clearSpool(notePath string) {
	_ = os.Remove(spoolPath(notePath))
}

// truncateSpool drops the oldest content once it exceeds maxSpoolBytes, cutting
// on a newline so the remainder stays valid UTF-8.
func truncateSpool(content string) string {
	if len(content) <= maxSpoolBytes {
		return content
	}
	cut := len(content) - maxSpoolBytes
	if i := strings.IndexByte(content[cut:], '\n'); i >= 0 {
		cut += i + 1
	}
	return spoolTruncatedNotice + content[cut:]
}

// flushPendingAppends writes any spooled content for notePath into the note.
// Used on paths that rewrite the note wholesale, so the pending content lands
// before the rewrite reads it rather than after.
func flushPendingAppends(notePath string) {
	pending := readSpool(notePath)
	if pending == "" {
		return
	}
	n, err := rawAppend(notePath, pending)
	if err != nil {
		_ = writeSpool(notePath, pending[n:])
		return
	}
	clearSpool(notePath)
}
