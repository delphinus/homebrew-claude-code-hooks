package main

import (
	"fmt"
	"os"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/backfill"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/notify"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/save"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/setup"
)

const usage = `Usage: claude-code-hooks <command> [args]

Commands:
  save              Save Claude Code conversation to Obsidian (reads JSON from stdin)
  backfill [--dry-run]  Backfill related links between session notes
  notify TITLE MSG  Show macOS notification (suppressed if WezTerm pane is focused)
  setup [--diff]    Merge hooks.json into ~/.claude/settings.json
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "save":
		input, e := hookdata.ReadFromStdin()
		if e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
		err = save.Run(input)

	case "backfill":
		dryRun := len(os.Args) > 2 && os.Args[2] == "--dry-run"
		err = backfill.Run(dryRun)

	case "notify":
		title := "Claude Code"
		message := "User interaction required"
		if len(os.Args) > 2 {
			title = os.Args[2]
		}
		if len(os.Args) > 3 {
			message = os.Args[3]
		}
		err = notify.Run(title, message)

	case "setup":
		diffMode := len(os.Args) > 2 && os.Args[2] == "--diff"
		err = setup.Run(diffMode)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
