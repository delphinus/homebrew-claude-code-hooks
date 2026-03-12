package main

import (
	"fmt"
	"os"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/backfill"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/completion"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/notify"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/save"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/setup"
)

var version = "dev"

const usage = `Usage: claude-code-hooks <command> [args]

Commands:
  save              Save Claude Code conversation to Obsidian (reads JSON from stdin)
  backfill [--dry-run]  Backfill related links between session notes
  notify TITLE MSG  Show macOS notification (suppressed if WezTerm pane is focused)
  setup [--diff]    Merge hooks.json into ~/.claude/settings.json
  completion SHELL  Output shell completion script (bash, zsh, fish)

Flags:
  --version, -v     Show version

Optional:
  Obsidian の Advanced URI プラグインを導入すると、ノートを新しいタブで開きます。
  未導入の場合は既存のタブで開きます。
  https://github.com/Vinzent03/obsidian-advanced-uri
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "-v":
		fmt.Println(version)
		return
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

	case "_session-end-bg":
		if len(os.Args) < 3 {
			os.Exit(1)
		}
		save.RunSessionEndBG(os.Args[2])
		return

	case "completion":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: claude-code-hooks completion <bash|zsh|fish>")
			os.Exit(1)
		}
		script, e := completion.Script(os.Args[2])
		if e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
		fmt.Print(script)

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
