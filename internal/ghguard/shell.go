package ghguard

import "strings"

// segments splits a shell command line into the individual commands it runs.
//
// This is deliberately not a shell parser. It needs to answer one question —
// "is there a gh invocation here, and with which arguments?" — so it only has
// to get quoting and command boundaries right. Anything it cannot interpret
// (parameter expansion, here-documents) stays in the word it appeared in, which
// at worst makes a command look unfamiliar and therefore a write.
//
// Command substitutions are treated as boundaries rather than skipped, because
// `echo "$(gh pr create ...)"` really does create the pull request.
func segments(command string) [][]string {
	var (
		all  [][]string
		cur  []string
		buf  strings.Builder
		word bool
	)
	flush := func() {
		if word {
			cur = append(cur, buf.String())
			buf.Reset()
			word = false
		}
	}
	brk := func() {
		flush()
		if len(cur) > 0 {
			all = append(all, cur)
			cur = nil
		}
	}
	// put appends to the current word. A word exists only once it has a rune in
	// it, so an empty quoted argument ("") is dropped — harmless here, since
	// only the leading positionals of a gh command are ever read.
	put := func(r rune) {
		buf.WriteRune(r)
		word = true
	}

	rs := []rune(command)
	dquote := false
	for i := 0; i < len(rs); i++ {
		c := rs[i]

		if dquote {
			switch {
			case c == '"':
				dquote = false
			case c == '\\' && i+1 < len(rs):
				i++
				put(rs[i])
			case c == '$' && i+1 < len(rs) && rs[i+1] == '(':
				// The substitution runs a command of its own, so leave the
				// quoted context and let its words split normally.
				i++
				dquote = false
				brk()
			default:
				put(c)
			}
			continue
		}

		switch {
		case c == '\\' && i+1 < len(rs):
			i++
			put(rs[i])

		case c == '\'':
			for i++; i < len(rs) && rs[i] != '\''; i++ {
				put(rs[i])
			}

		case c == '"':
			dquote = true

		case c == '$' && i+1 < len(rs) && rs[i+1] == '(':
			i++
			brk()

		case c == '`' || c == '(' || c == ')' || c == '{' || c == '}' ||
			c == ';' || c == '\n' || c == '<' || c == '>':
			brk()

		case c == '&' || c == '|':
			if i+1 < len(rs) && rs[i+1] == c {
				i++
			}
			brk()

		case c == ' ' || c == '\t' || c == '\r':
			flush()

		default:
			put(c)
		}
	}
	brk()
	return all
}
