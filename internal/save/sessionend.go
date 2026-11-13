package save

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/config"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/frontmatter"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
	"github.com/delphinus/homebrew-claude-code-hooks/internal/note"
)

// sessionEndSync controls whether SessionEnd runs inline (for tests)
// or spawns a detached background process (production).
var sessionEndSync bool

// handleSessionEnd spawns a detached background process to generate
// summaries and titles, then returns immediately so the hook exits
// before Claude Code kills it during shutdown.
func handleSessionEnd(input *hookdata.HookInput) error {
	cacheDir := config.CacheDir()
	sessionCachePath := filepath.Join(cacheDir, input.SessionID)

	if _, err := os.Stat(sessionCachePath); err != nil {
		return nil
	}

	if sessionEndSync {
		RunSessionEndBG(input.SessionID, input.TranscriptPath)
		return nil
	}

	// Spawn a detached background process to do the heavy work
	// (calling claude -p for summaries/titles). The hook returns
	// immediately so Claude Code won't kill us during shutdown.
	exe, err := os.Executable()
	if err != nil {
		return nil
	}

	args := []string{"_session-end-bg", input.SessionID}
	if input.TranscriptPath != "" {
		args = append(args, input.TranscriptPath)
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return nil
	}

	// Detach: don't wait for the child
	cmd.Process.Release()
	return nil
}

// RunSessionEndBG is the background worker for SessionEnd processing.
// It is invoked as: claude-code-hooks _session-end-bg <session_id> [transcript_path]
func RunSessionEndBG(sessionID, transcriptPath string) {
	// Ignore signals so we survive even if parent is killed
	signal.Ignore(syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	cacheDir := config.CacheDir()
	sessionCachePath := filepath.Join(cacheDir, sessionID)
	vaultDir := config.VaultDir()

	sessionNotes, err := note.FindNotesBySessionID(sessionID, vaultDir)
	if err != nil || len(sessionNotes) == 0 {
		os.Remove(sessionCachePath)
		return
	}

	// The notes only carry HH:MM:SS. The transcript has full timestamps, which
	// pin down the date when the session was resumed after a day or more.
	transcript := transcriptTimes(transcriptPath)

	for _, notePath := range sessionNotes {
		if _, err := os.Stat(notePath); err != nil {
			continue
		}

		flushCachedPlan(sessionID, notePath)
		// Land anything a failed append left in the spool before the note is
		// read and rewritten below, so it does not end up after the summary.
		flushPendingAppends(notePath)

		noteContent, err := readHead(notePath, 100000)
		if err != nil {
			continue
		}

		// Record the session end time (= last activity) into frontmatter.
		// "ended" is the last timestamped entry in the note, which tracks real
		// work end without the idle inflation of the actual quit time. The quit
		// time is used only when the note has no timestamped entry at all.
		start := ""
		if fm, _, perr := frontmatter.Parse(noteContent); perr == nil {
			start = fm.Date
		}
		ended := lastActivityTime(notePath, start, transcript)
		if ended == "" {
			ended = time.Now().Format("2006-01-02T15:04:05")
		}
		setEnded(notePath, ended)

		summary := claudeGenerate(
			"以下の Claude Code の会話ログを日本語で3〜5行で要約してください。要約のみを出力し、前置きは不要です。",
			noteContent,
		)

		if summary != "" {
			// Prepend a human-readable time line to the summary callout.
			if tl := summaryTimeLine(start, ended); tl != "" {
				summary = tl + "\n" + summary
			}
			insertSummary(notePath, summary)
		}

		basename := strings.TrimSuffix(filepath.Base(notePath), ".md")
		parts := strings.SplitN(basename, "-", 4)
		if len(parts) < 4 {
			continue
		}
		prefix := strings.Join(parts[:3], "-")
		currentTitle := parts[3]
		project := filepath.Base(filepath.Dir(notePath))

		if currentTitle == project {
			newTitle := claudeGenerate(
				"以下の Claude Code の会話ログの内容を表す短いタイトル（50文字以内）を日本語で生成してください。タイトルのみを出力し、前置きや引用符は不要です。",
				noteContent,
			)

			if newTitle != "" {
				renameNote(notePath, prefix, newTitle, vaultDir)
			}
		}
	}

	os.Remove(sessionCachePath)
	os.Remove(filepath.Join(cacheDir, sessionID+"-last-msg"))
}

func readHead(path string, maxBytes int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	return string(buf[:n]), nil
}

// activityTSRe matches every timestamped entry the hooks write: the
// "## 🤖 Assistant (HH:MM:SS)" headings (the emoji is optional so notes written
// before it was added still match) and the callout headers such as
// "> [!question] User (HH:MM:SS)" and "> [!terminal]- <description> (HH:MM:SS)".
// A note split off by a cwd change can consist of callouts only, so the
// Assistant headings alone are not enough to find its last activity.
var activityTSRe = regexp.MustCompile(`(?m)^(?:## (?:🤖 )?Assistant|> \[![a-z]+\][-+]?.*) \((\d{2}):(\d{2}):(\d{2})\)[ \t]*$`)

// rollbackTolerance is how far a timestamp may go backwards before it is taken
// as the next day. Entries are appended in order, but a spooled append can land
// slightly late; a real midnight crossing goes back by many hours.
const rollbackTolerance = time.Hour

// lastActivityTime returns the time of the last timestamped entry in the note
// (Assistant heading or callout header), as "2006-01-02T15:04:05". The entries
// only carry HH:MM:SS, so each one is dated by walking them in order from
// dateStr (frontmatter "date"):
//
//   - If the transcript has a timestamp at the same HH:MM (within a minute) at
//     or after the current position, its date is used. This is what gets a
//     session resumed after a day or more right.
//   - Otherwise a day is added whenever the clock goes backwards, which only
//     handles gaps shorter than a day.
//
// This is the last real activity, used as the session end time so that a
// session left open and closed later does not count the idle time.
// Returns "" if the note has no timestamped entry or dateStr is unparseable.
func lastActivityTime(notePath, dateStr string, transcript []time.Time) string {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return ""
	}
	matches := activityTSRe.FindAllStringSubmatch(string(content), -1)
	if len(matches) == 0 {
		return ""
	}

	const layout = "2006-01-02T15:04:05"
	start, err := time.ParseInLocation(layout, dateStr, time.Local)
	if err != nil {
		return ""
	}
	cur, last := start, start
	for _, m := range matches {
		hh, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		ss, _ := strconv.Atoi(m[3])
		var t time.Time
		if y, mo, d, ok := transcriptDate(transcript, cur, hh, mm); ok {
			t = time.Date(y, mo, d, hh, mm, ss, 0, time.Local)
			if t.Before(cur) {
				continue
			}
		} else {
			t = time.Date(cur.Year(), cur.Month(), cur.Day(), hh, mm, ss, 0, time.Local)
			if t.Before(cur.Add(-rollbackTolerance)) {
				t = t.AddDate(0, 0, 1)
			} else if t.Before(cur) {
				continue // slightly out of order; keep the later time
			}
		}
		cur = t
		if t.After(last) {
			last = t
		}
	}
	return last.Format(layout)
}

// transcriptDate looks for the first transcript timestamp at or after
// cur - rollbackTolerance whose clock reads hh:mm (within a minute), and
// returns its date.
func transcriptDate(transcript []time.Time, cur time.Time, hh, mm int) (int, time.Month, int, bool) {
	lo := cur.Add(-rollbackTolerance)
	i := sort.Search(len(transcript), func(i int) bool { return !transcript[i].Before(lo) })
	want := hh*60 + mm
	for _, t := range transcript[i:] {
		got := t.Hour()*60 + t.Minute()
		diff := got - want
		if diff < 0 {
			diff = -diff
		}
		if diff <= 1 || diff == 24*60-1 {
			if t.Hour() == 0 && hh == 23 {
				t = t.AddDate(0, 0, -1) // 23:59 in the note, 00:00 in the transcript
			} else if t.Hour() == 23 && hh == 0 {
				t = t.AddDate(0, 0, 1)
			}
			return t.Year(), t.Month(), t.Day(), true
		}
	}
	return 0, 0, 0, false
}

// transcriptTimes reads every "timestamp" in the transcript (JSON Lines) and
// returns them in local time, sorted. A missing or unreadable transcript gives
// nil, and the caller falls back to walking the clock times.
func transcriptTimes(path string) []time.Time {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []time.Time
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if m := transcriptTSRe.FindSubmatch(line); m != nil {
			if t, perr := time.Parse(time.RFC3339Nano, string(m[1])); perr == nil {
				out = append(out, t.In(time.Local))
			}
		}
		if err != nil {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

var transcriptTSRe = regexp.MustCompile(`"timestamp":"([^"]+)"`)

// setEnded writes the ended timestamp into the note's frontmatter.
func setEnded(notePath, ended string) {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return
	}
	fm, body, err := frontmatter.Parse(string(content))
	if err != nil {
		return
	}
	fm.Ended = ended
	os.WriteFile(notePath, []byte(fm.Render()+body), 0o644)
}

// summaryTimeLine renders a "⏱ HH:MM–HH:MM (Xh Ym)" line for the summary callout
// from start/ended (both "2006-01-02T15:04:05"). Returns "" if either is
// unparseable.
func summaryTimeLine(start, ended string) string {
	const layout = "2006-01-02T15:04:05"
	s, err1 := time.ParseInLocation(layout, start, time.Local)
	e, err2 := time.ParseInLocation(layout, ended, time.Local)
	if err1 != nil || err2 != nil {
		return ""
	}
	d := e.Sub(s)
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	cross := ""
	sd := time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, time.Local)
	ed := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, time.Local)
	if days := int(ed.Sub(sd).Hours()/24 + 0.5); days > 0 {
		cross = fmt.Sprintf("(+%dd)", days)
	}
	return fmt.Sprintf("⏱ %s–%s%s (%dh%02dm)", s.Format("15:04"), e.Format("15:04"), cross, h, m)
}

// claudeGenerate runs `claude -p` to generate a summary/title from a conversation
// log. It is a package var so tests can stub it out (avoiding a real, slow,
// non-deterministic network call to the claude CLI).
var claudeGenerate = func(prompt, content string) string {
	cmd := exec.Command("claude", "-p", "--model", "haiku", "--setting-sources", "")
	cmd.Stdin = strings.NewReader(content)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Filter out CLAUDECODE to avoid "nested session" rejection.
	// --setting-sources "" prevents loading hooks config, so the inner
	// claude -p session won't fire its own SessionEnd hook.
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env

	cmd.Args = append(cmd.Args, prompt)

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func buildSummaryBlock(summary string) string {
	var b strings.Builder
	b.WriteString("> [!summary]\n")
	for _, line := range strings.Split(summary, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	return b.String()
}

func insertSummary(notePath, summary string) {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return
	}

	text := string(content)
	summaryBlock := buildSummaryBlock(summary)

	// Replace existing summary if present
	if start := strings.Index(text, "> [!summary]\n"); start >= 0 {
		// Find the end of the summary block (consecutive "> " lines)
		end := start + len("> [!summary]\n")
		for end < len(text) {
			lineEnd := strings.Index(text[end:], "\n")
			if lineEnd < 0 {
				end = len(text)
				break
			}
			line := text[end : end+lineEnd]
			if !strings.HasPrefix(line, "> ") {
				break
			}
			end += lineEnd + 1
		}

		var b strings.Builder
		b.WriteString(text[:start])
		b.WriteString(summaryBlock)
		b.WriteString(text[end:])
		os.WriteFile(notePath, []byte(b.String()), 0o644)
		return
	}

	// Find the end of frontmatter (second ---)
	idx := strings.Index(text, "---\n")
	if idx < 0 {
		return
	}
	rest := text[idx+4:]
	endFM := strings.Index(rest, "---\n")
	if endFM < 0 {
		return
	}

	// Position right after closing ---
	insertPos := idx + 4 + endFM + 4

	var b strings.Builder
	b.WriteString(text[:insertPos])
	b.WriteString("\n")
	b.WriteString(summaryBlock)
	b.WriteString("\n")
	b.WriteString(text[insertPos:])

	os.WriteFile(notePath, []byte(b.String()), 0o644)
}

func renameNote(notePath, prefix, newTitle, vaultDir string) {
	oldBasename := strings.TrimSuffix(filepath.Base(notePath), ".md")

	fileTitle := note.MakeFilenameTitle(newTitle)
	displayTitle := note.MakeDisplayTitle(newTitle)
	idSlug := note.MakeIDSlug(fileTitle)

	if fileTitle == "" {
		return
	}

	newNotePath := filepath.Join(filepath.Dir(notePath), prefix+"-"+fileTitle+".md")
	newID := prefix + "-" + idSlug
	newBasename := prefix + "-" + fileTitle

	// Update frontmatter id and aliases
	content, err := os.ReadFile(notePath)
	if err != nil {
		return
	}

	text := string(content)

	// Update id field
	fm, body, err := frontmatter.Parse(text)
	if err != nil {
		return
	}

	fm.ID = newID
	if len(fm.Aliases) > 0 {
		fm.Aliases[0] = displayTitle
	} else {
		fm.Aliases = []string{displayTitle}
	}

	os.WriteFile(notePath, []byte(fm.Render()+body), 0o644)

	// Rename the file
	if err := os.Rename(notePath, newNotePath); err != nil {
		return
	}

	// Keep the plan re-post cache pointing at the new path if it referenced this
	// note. flushCachedPlan runs before the rename and stores the old path, so
	// without this the "Plan from" link in a resumed session can't be resolved.
	updatePlanLastPath(notePath, newNotePath)

	// Update references in other files
	updateReferences(oldBasename, newBasename, newNotePath, vaultDir)
}

// updatePlanLastPath rewrites the note path recorded in the "<project>-plan-last"
// re-post cache when a note is renamed, so the original-note link survives.
func updatePlanLastPath(oldPath, newPath string) {
	cacheDir := config.CacheDir()
	project := filepath.Base(filepath.Dir(newPath))
	planLastPath := filepath.Join(cacheDir, project+"-plan-last")

	data, err := os.ReadFile(planLastPath)
	if err != nil {
		return
	}
	parts := strings.SplitN(string(data), "\n", 2)
	if len(parts) < 2 || parts[0] != oldPath {
		return
	}
	_ = os.WriteFile(planLastPath, []byte(newPath+"\n"+parts[1]), 0o644)
}

func updateReferences(oldName, newName, excludePath, vaultDir string) {
	oldLink := "[[" + oldName + "]]"
	newLink := "[[" + newName + "]]"

	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == excludePath || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		text := string(content)
		if !strings.Contains(text, oldLink) {
			return nil
		}

		newText := strings.ReplaceAll(text, oldLink, newLink)
		os.WriteFile(path, []byte(newText), 0o644)
		return nil
	})
}
