// Package opencmd implements the "open" subcommand: opening a session note in
// Obsidian on demand (replacing the automatic open at session start).
package opencmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

const defaultListLimit = 20

// Run handles `claude-code-hooks open [args]`.
//
//	open                 open the current session's note (most recently modified)
//	open --list [N]      print recent session notes as JSON (default 20)
//	open <session-id>    open the note for a session id (full or prefix)
//	open <path.md>       open a specific note file
func Run(args []string) error {
	if len(args) > 0 && args[0] == "--list" {
		limit := defaultListLimit
		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid limit %q: %w", args[1], err)
			}
			limit = n
		}
		return runList(limit)
	}

	var (
		target string
		err    error
	)
	switch {
	case len(args) == 0:
		cwd, _ := os.Getwd()
		target, err = note.MostRecentNoteForCWD(cwd)
	case strings.HasSuffix(args[0], ".md") && fileExists(args[0]):
		target = args[0]
	default:
		target, err = note.ResolveBySessionID(args[0])
	}
	if err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("no note found")
	}
	if err := note.OpenNote(target); err != nil {
		return fmt.Errorf("opening note: %w", err)
	}
	fmt.Println(target)
	return nil
}

func runList(limit int) error {
	metas, err := note.RecentNotes(limit)
	if err != nil {
		return err
	}
	if metas == nil {
		metas = []note.NoteMeta{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(metas)
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
