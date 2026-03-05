_claude_code_hooks() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "$prev" in
        claude-code-hooks)
            COMPREPLY=($(compgen -W "save backfill notify setup completion" -- "$cur"))
            return
            ;;
        backfill)
            COMPREPLY=($(compgen -W "--dry-run" -- "$cur"))
            return
            ;;
        setup)
            COMPREPLY=($(compgen -W "--diff" -- "$cur"))
            return
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            return
            ;;
    esac
}

complete -F _claude_code_hooks claude-code-hooks
