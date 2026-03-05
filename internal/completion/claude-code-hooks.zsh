#compdef claude-code-hooks

_claude_code_hooks() {
    local -a commands
    commands=(
        'save:Save Claude Code conversation to Obsidian'
        'backfill:Backfill related links between session notes'
        'notify:Show macOS notification'
        'setup:Merge hooks.json into settings.json'
        'completion:Output shell completion script'
    )

    _arguments -C \
        '1:command:->command' \
        '*::arg:->args'

    case "$state" in
        command)
            _describe 'command' commands
            ;;
        args)
            case "${words[1]}" in
                backfill)
                    _arguments '--dry-run[Show what would be changed without modifying files]'
                    ;;
                setup)
                    _arguments '--diff[Show diff without applying changes]'
                    ;;
                completion)
                    _values 'shell' bash zsh fish
                    ;;
            esac
            ;;
    esac
}

_claude_code_hooks "$@"
