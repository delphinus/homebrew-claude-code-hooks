package note

import (
	"crypto/rand"
	"regexp"
	"strings"
)

var (
	invalidChars    = regexp.MustCompile(`[/\\:*?"<>|#\[\]{}'` + "`]")
	multipleSpaces  = regexp.MustCompile(`\s+`)
	multipleHyphens = regexp.MustCompile(`-+`)
	nonASCIISlug    = regexp.MustCompile(`[^a-z0-9-]`)
)

// MakeDisplayTitle creates a human-readable title from the first line of a prompt.
// Invalid filename characters are removed but spaces are preserved.
func MakeDisplayTitle(prompt string) string {
	line := firstLine(prompt)
	if len([]rune(line)) > 50 {
		line = string([]rune(line)[:50])
	}
	line = invalidChars.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)
	return line
}

// MakeFilenameTitle creates a kebab-case filename from the first line of a prompt.
func MakeFilenameTitle(prompt string) string {
	line := firstLine(prompt)
	if len([]rune(line)) > 50 {
		line = string([]rune(line)[:50])
	}
	line = invalidChars.ReplaceAllString(line, "")
	line = multipleSpaces.ReplaceAllString(line, "-")
	line = multipleHyphens.ReplaceAllString(line, "-")
	line = strings.Trim(line, "-")
	return line
}

// MakeIDSlug creates an ASCII-only slug for use in note IDs.
// If the result is empty (e.g., all non-ASCII input), generates a random 4-character string.
func MakeIDSlug(text string) string {
	slug := strings.ToLower(text)
	slug = nonASCIISlug.ReplaceAllString(slug, "")
	slug = multipleHyphens.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return randomChars(4)
	}
	return slug
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func randomChars(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "XXXX"
	}
	for i := range b {
		b[i] = letters[b[i]%byte(len(letters))]
	}
	return string(b)
}
