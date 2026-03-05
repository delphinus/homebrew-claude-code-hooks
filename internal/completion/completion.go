package completion

import (
	_ "embed"
	"fmt"
)

//go:embed claude-code-hooks.bash
var bashScript string

//go:embed claude-code-hooks.zsh
var zshScript string

//go:embed claude-code-hooks.fish
var fishScript string

// Script returns the completion script for the given shell.
func Script(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashScript, nil
	case "zsh":
		return zshScript, nil
	case "fish":
		return fishScript, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
}
