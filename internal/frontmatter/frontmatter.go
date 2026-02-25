package frontmatter

import (
	"fmt"
	"strings"
)

// Frontmatter represents the YAML frontmatter of an Obsidian note.
type Frontmatter struct {
	ID        string   `yaml:"id"`
	Aliases   []string `yaml:"aliases"`
	Tags      []string `yaml:"tags"`
	Date      string   `yaml:"date"`
	SessionID string   `yaml:"session_id"`
	Hostname  string   `yaml:"hostname"`
	CWD       string   `yaml:"cwd"`
	Related   []string `yaml:"related,omitempty"`
}

// Parse splits a note into its frontmatter and body.
// Returns the parsed Frontmatter, the body text (after closing ---), and any error.
func Parse(content string) (*Frontmatter, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content, fmt.Errorf("no frontmatter found")
	}

	rest := content[4:] // skip opening "---\n"
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		// Check for --- at end of file
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - 3
		} else {
			return nil, content, fmt.Errorf("unclosed frontmatter")
		}
	}

	yamlStr := rest[:end]
	body := rest[end+4:] // skip "\n---\n"
	if strings.HasSuffix(rest, "\n---") && end == len(rest)-3 {
		body = ""
	}

	fm := &Frontmatter{}
	if err := parseYAML(yamlStr, fm); err != nil {
		return nil, content, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return fm, body, nil
}

// parseYAML is a minimal hand-parser for the known frontmatter fields.
// We parse manually to avoid issues with yaml.v3 reordering and to keep control.
func parseYAML(s string, fm *Frontmatter) error {
	lines := strings.Split(s, "\n")
	var currentList *[]string

	for _, line := range lines {
		if strings.HasPrefix(line, "  - ") {
			if currentList != nil {
				val := strings.TrimPrefix(line, "  - ")
				*currentList = append(*currentList, val)
			}
			continue
		}

		currentList = nil

		idx := strings.Index(line, ": ")
		if idx < 0 {
			// could be a key with no value followed by list items
			if strings.HasSuffix(line, ":") {
				key := strings.TrimSuffix(line, ":")
				switch key {
				case "aliases":
					currentList = &fm.Aliases
				case "tags":
					currentList = &fm.Tags
				case "related":
					currentList = &fm.Related
				}
			}
			continue
		}

		key := line[:idx]
		val := line[idx+2:]

		switch key {
		case "id":
			fm.ID = val
		case "date":
			fm.Date = val
		case "session_id":
			fm.SessionID = val
		case "hostname":
			fm.Hostname = val
		case "cwd":
			fm.CWD = val
		}
	}

	return nil
}

// Render outputs the frontmatter in the fixed field order with --- delimiters.
func (f *Frontmatter) Render() string {
	var b strings.Builder
	b.WriteString("---\n")

	fmt.Fprintf(&b, "id: %s\n", f.ID)

	b.WriteString("aliases:\n")
	for _, a := range f.Aliases {
		fmt.Fprintf(&b, "  - %s\n", a)
	}

	b.WriteString("tags:\n")
	for _, t := range f.Tags {
		fmt.Fprintf(&b, "  - %s\n", t)
	}

	fmt.Fprintf(&b, "date: %s\n", f.Date)
	fmt.Fprintf(&b, "session_id: %s\n", f.SessionID)
	fmt.Fprintf(&b, "hostname: %s\n", f.Hostname)
	fmt.Fprintf(&b, "cwd: %s\n", f.CWD)

	if len(f.Related) > 0 {
		b.WriteString("related:\n")
		for _, r := range f.Related {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}

	b.WriteString("---\n")
	return b.String()
}
