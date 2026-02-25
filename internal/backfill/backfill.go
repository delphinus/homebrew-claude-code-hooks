package backfill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
)

// Run executes the backfill subcommand.
// It finds all notes sharing a session_id and adds missing related links.
func Run(dryRun bool) error {
	vaultDir := config.VaultDir()

	if _, err := os.Stat(vaultDir); err != nil {
		return fmt.Errorf("vault not found: %s", vaultDir)
	}

	// Build session_id → []filepath mapping
	sidMap := make(map[string][]string)

	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		sid := extractSessionID(path)
		if sid != "" {
			sidMap[sid] = append(sidMap[sid], path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning vault: %w", err)
	}

	updated := 0
	skipped := 0

	for sid, files := range sidMap {
		if len(files) < 2 {
			continue
		}

		sidShort := sid
		if len(sidShort) > 8 {
			sidShort = sidShort[:8]
		}
		fmt.Printf("Session %s...: %d notes\n", sidShort, len(files))

		for _, file := range files {
			bn := filepath.Base(file)

			// Collect other note names
			var others []string
			for _, other := range files {
				if other == file {
					continue
				}
				others = append(others, strings.TrimSuffix(filepath.Base(other), ".md"))
			}

			// Find missing links
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			text := string(content)

			var missing []string
			for _, name := range others {
				if !strings.Contains(text, "[["+name+"]]") {
					missing = append(missing, name)
				}
			}

			if len(missing) == 0 {
				skipped++
				continue
			}

			fmt.Printf("  [update] %s (+%d link(s))\n", bn, len(missing))
			for _, name := range missing {
				fmt.Printf("    + [[%s]]\n", name)
			}

			if dryRun {
				updated++
				continue
			}

			// Update frontmatter related field
			if err := updateFrontmatterRelated(file, missing); err != nil {
				fmt.Printf("    [warn] frontmatter update failed: %v\n", err)
				continue
			}

			// Update body [!link] section
			if err := updateBodyLinks(file, missing); err != nil {
				fmt.Printf("    [warn] body link update failed: %v\n", err)
				continue
			}

			updated++
		}
	}

	fmt.Println()
	if dryRun {
		fmt.Printf("[dry-run] %d notes would be updated, %d already linked\n", updated, skipped)
	} else {
		fmt.Printf("Complete: %d updated, %d already linked\n", updated, skipped)
	}

	return nil
}

func extractSessionID(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "session_id: ") {
			return strings.TrimPrefix(line, "session_id: ")
		}
	}
	return ""
}

func updateFrontmatterRelated(file string, missing []string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	text := string(content)

	if strings.Contains(text, "related:") {
		// Append to existing related section
		var newEntries strings.Builder
		for _, name := range missing {
			fmt.Fprintf(&newEntries, "  - \"[[%s]]\"\n", name)
		}

		// Find the end of the related section and insert before the next non-list line
		lines := strings.Split(text, "\n")
		var result []string
		inRelated := false
		inserted := false

		for _, line := range lines {
			if line == "related:" {
				inRelated = true
				result = append(result, line)
				continue
			}
			if inRelated && !strings.HasPrefix(line, "  - ") {
				// End of related section, insert new entries
				for _, name := range missing {
					result = append(result, fmt.Sprintf("  - \"[[%s]]\"", name))
				}
				inRelated = false
				inserted = true
			}
			result = append(result, line)
		}
		// If related was the last section before ---
		if inRelated && !inserted {
			for _, name := range missing {
				result = append(result, fmt.Sprintf("  - \"[[%s]]\"", name))
			}
		}

		return os.WriteFile(file, []byte(strings.Join(result, "\n")), 0o644)
	}

	// Add new related field before closing ---
	fm, body, err := frontmatter.Parse(text)
	if err != nil {
		return err
	}

	for _, name := range missing {
		fm.Related = append(fm.Related, fmt.Sprintf(`"[[%s]]"`, name))
	}

	return os.WriteFile(file, []byte(fm.Render()+body), 0o644)
}

func updateBodyLinks(file string, missing []string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	text := string(content)

	if strings.Contains(text, "> [!link]") {
		// Append to existing link section
		lines := strings.Split(text, "\n")
		var result []string
		inLink := false
		inserted := false

		for i, line := range lines {
			if strings.HasPrefix(line, "> [!link]") {
				inLink = true
				result = append(result, line)
				continue
			}
			if inLink && line == "" {
				// End of link section, insert before empty line
				for _, name := range missing {
					result = append(result, fmt.Sprintf("> - [[%s]]", name))
				}
				inLink = false
				inserted = true
			}
			result = append(result, line)
			_ = i
		}
		if inLink && !inserted {
			for _, name := range missing {
				result = append(result, fmt.Sprintf("> - [[%s]]", name))
			}
		}

		return os.WriteFile(file, []byte(strings.Join(result, "\n")), 0o644)
	}

	// Insert new link section after frontmatter
	idx := strings.Index(text, "---\n")
	if idx < 0 {
		return nil
	}
	rest := text[idx+4:]
	endFM := strings.Index(rest, "---\n")
	if endFM < 0 {
		return nil
	}

	insertPos := idx + 4 + endFM + 4

	var b strings.Builder
	b.WriteString(text[:insertPos])
	b.WriteString("\n> [!link] Previous Sessions\n")
	for _, name := range missing {
		fmt.Fprintf(&b, "> - [[%s]]\n", name)
	}
	b.WriteString("\n")
	b.WriteString(text[insertPos:])

	return os.WriteFile(file, []byte(b.String()), 0o644)
}
