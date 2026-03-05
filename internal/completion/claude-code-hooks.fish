complete -c claude-code-hooks -f

complete -c claude-code-hooks -n __fish_use_subcommand -a save -d 'Save Claude Code conversation to Obsidian'
complete -c claude-code-hooks -n __fish_use_subcommand -a backfill -d 'Backfill related links between session notes'
complete -c claude-code-hooks -n __fish_use_subcommand -a notify -d 'Show macOS notification'
complete -c claude-code-hooks -n __fish_use_subcommand -a setup -d 'Merge hooks.json into settings.json'
complete -c claude-code-hooks -n __fish_use_subcommand -a completion -d 'Output shell completion script'

complete -c claude-code-hooks -n '__fish_seen_subcommand_from backfill' -l dry-run -d 'Show what would be changed without modifying files'
complete -c claude-code-hooks -n '__fish_seen_subcommand_from setup' -l diff -d 'Show diff without applying changes'
complete -c claude-code-hooks -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish' -d 'Shell type'
