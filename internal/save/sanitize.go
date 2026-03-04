package save

import "strings"

// EnsureTableBlankLines inserts blank lines before and after Markdown table
// blocks when they are missing. Obsidian requires blank lines around tables
// for correct rendering, even though the CommonMark spec does not.
func EnsureTableBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines)+10)
	inCodeFence := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track code fences (``` or more backticks).
		wasInCodeFence := inCodeFence
		if strings.HasPrefix(trimmed, "```") {
			inCodeFence = !inCodeFence
		}

		// Skip blank-line logic for lines inside code fences and for
		// fence delimiters themselves (opening or closing).
		if !inCodeFence && !wasInCodeFence {
			isTableLine := strings.HasPrefix(trimmed, "|")

			// Insert blank line before a table when the previous line is
			// non-empty and not itself a table row.
			if isTableLine && i > 0 {
				prev := strings.TrimSpace(lines[i-1])
				if prev != "" && !strings.HasPrefix(prev, "|") {
					result = append(result, "")
				}
			}

			// Insert blank line after a table when the current line is
			// non-empty and not a table row, but the previous line was.
			if !isTableLine && trimmed != "" && i > 0 {
				prev := strings.TrimSpace(lines[i-1])
				if strings.HasPrefix(prev, "|") {
					result = append(result, "")
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
