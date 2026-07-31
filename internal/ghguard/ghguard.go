// Package ghguard inspects Bash commands from a PreToolUse hook and forces a
// permission prompt before `gh` writes to a guarded host.
//
// Claude Code's auto mode lets a classifier approve tool calls silently. That is
// fine for local work, but posting an issue comment or opening a pull request is
// outward-facing and effectively public the moment it happens. A PreToolUse hook
// that answers "ask" is the only mechanism that always reaches the user: the
// classifier may still deny such a call, but it cannot approve it without a
// prompt.
//
// Writes to other hosts (a self-hosted GitHub Enterprise, say) pass through
// untouched, so day-to-day work against an internal host is not interrupted.
// Which hosts are guarded is configurable; see GuardedHosts.
package ghguard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/delphinus/homebrew-claude-code-hooks/internal/hookdata"
)

const (
	// hostsEnv overrides the guarded host list (comma separated).
	hostsEnv = "CLAUDE_GH_GUARD_HOSTS"
	// defaultHost is guarded unless hostsEnv says otherwise. Only the public
	// host is guarded by default: it is the one where a mistake is immediately
	// visible to strangers and cannot be taken back.
	defaultHost = "github.com"
	// gitTimeout bounds the `git remote get-url` fallback. A hook runs in the
	// critical path of every gh command, so it must not hang.
	gitTimeout = 2 * time.Second
)

// writeCall describes a gh invocation that writes to a guarded host.
type writeCall struct {
	host    string
	repo    string
	summary string
}

type hookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

type decision struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

// Run reads a PreToolUse payload and prints an "ask" decision when the command
// writes to a guarded host. Anything else produces no output at all, which
// leaves the normal permission flow (including the auto mode classifier) in
// charge.
func Run(input *hookdata.HookInput) error {
	if input == nil || input.ToolName != "Bash" {
		return nil
	}
	call := findWrite(input.ToolInput.Command, input.CWD)
	if call == nil {
		return nil
	}
	return emit(os.Stdout, *call)
}

func emit(w io.Writer, call writeCall) error {
	target := call.host
	if call.repo != "" {
		target = fmt.Sprintf("%s の %s", call.host, call.repo)
	}
	d := decision{hookSpecificOutput{
		HookEventName:      "PreToolUse",
		PermissionDecision: "ask",
		PermissionDecisionReason: fmt.Sprintf(
			"%s への書き込み操作です (%s)。実行してよいか確認してください。",
			target, call.summary,
		),
	}}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		return fmt.Errorf("writing decision: %w", err)
	}
	return nil
}

// GuardedHosts returns the set of hosts whose writes need confirmation.
func GuardedHosts() map[string]bool {
	raw := os.Getenv(hostsEnv)
	if strings.TrimSpace(raw) == "" {
		raw = defaultHost
	}
	hosts := map[string]bool{}
	for _, h := range strings.Split(raw, ",") {
		if h = normalizeHost(h); h != "" {
			hosts[h] = true
		}
	}
	return hosts
}

// findWrite returns the first gh write to a guarded host in command, or nil.
//
// The command may chain several invocations (`a && gh ... ; b`), so every
// segment is inspected. One match is enough: the prompt shows the whole command
// anyway, and the user decides on it as a unit.
func findWrite(command, cwd string) *writeCall {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	guarded := GuardedHosts()
	for _, seg := range segments(command) {
		if call := analyze(seg, cwd); call != nil && guarded[call.host] {
			return call
		}
	}
	return nil
}

// analyze inspects one command segment and reports the gh write it performs.
func analyze(words []string, cwd string) *writeCall {
	words, env := stripPrefixes(words)
	if len(words) == 0 || path.Base(words[0]) != "gh" {
		return nil
	}

	args := parseArgs(words[1:])
	if !args.isWrite() {
		return nil
	}

	return &writeCall{
		host:    resolveHost(args, env, cwd),
		repo:    repoName(args.repo),
		summary: args.summary(),
	}
}

// resolveHost mirrors how gh itself picks a host, cheapest source first. The
// git remote is consulted last because it costs a subprocess.
func resolveHost(args ghArgs, env map[string]string, cwd string) string {
	for _, host := range []string{
		args.hostname,
		env["GH_HOST"],
		hostFromRepo(args.repo),
		os.Getenv("GH_HOST"),
	} {
		if host != "" {
			return normalizeHost(host)
		}
	}
	if host := hostFromGitRemote(cwd); host != "" {
		return normalizeHost(host)
	}
	return defaultHost
}

// stripPrefixes drops leading variable assignments and wrapper commands so that
// `GH_HOST=x env gh pr create` is recognised as a gh invocation. The
// assignments are returned because GH_HOST among them selects the host.
func stripPrefixes(words []string) ([]string, map[string]string) {
	env := map[string]string{}
	for len(words) > 0 {
		w := words[0]
		if name, value, ok := envAssignment(w); ok {
			env[name] = value
			words = words[1:]
			continue
		}
		switch path.Base(w) {
		case "env", "command", "nohup", "time", "builtin", "exec":
			words = words[1:]
			continue
		}
		break
	}
	return words, env
}

// envAssignment reports whether w is a NAME=VALUE shell assignment.
func envAssignment(w string) (string, string, bool) {
	i := strings.Index(w, "=")
	if i <= 0 {
		return "", "", false
	}
	for j, c := range w[:i] {
		alpha := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !alpha && !(j > 0 && c >= '0' && c <= '9') {
			return "", "", false
		}
	}
	return w[:i], w[i+1:], true
}

// ghArgs holds the parts of a gh command line the guard cares about.
type ghArgs struct {
	positional []string
	hostname   string
	repo       string
	method     string
	hasFields  bool
}

// parseArgs picks out the flags that decide host and write-ness. Values of
// flags we do not know are allowed to land in positional; only the first two
// entries are ever read, and those come from before such flags in practice.
func parseArgs(args []string) ghArgs {
	var g ghArgs
	// value returns the argument after i and advances past it.
	value := func(i *int) string {
		if *i+1 < len(args) {
			*i++
			return args[*i]
		}
		return ""
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--hostname":
			g.hostname = value(&i)
		case strings.HasPrefix(a, "--hostname="):
			g.hostname = strings.TrimPrefix(a, "--hostname=")
		case a == "--repo" || a == "-R":
			g.repo = value(&i)
		case strings.HasPrefix(a, "--repo="):
			g.repo = strings.TrimPrefix(a, "--repo=")
		case a == "--method" || a == "-X":
			g.method = value(&i)
		case strings.HasPrefix(a, "--method="):
			g.method = strings.TrimPrefix(a, "--method=")
		case a == "-f" || a == "-F" || a == "--field" || a == "--raw-field" || a == "--input":
			// gh api switches to POST as soon as a field is given.
			g.hasFields = true
			value(&i)
		case strings.HasPrefix(a, "-f") && len(a) > 2,
			strings.HasPrefix(a, "-F") && len(a) > 2,
			strings.HasPrefix(a, "--field="),
			strings.HasPrefix(a, "--raw-field="),
			strings.HasPrefix(a, "--input="):
			g.hasFields = true
		case strings.HasPrefix(a, "-"):
			// Unknown flag: leave it alone.
		default:
			g.positional = append(g.positional, a)
		}
	}
	return g
}

func (g ghArgs) sub() string {
	if len(g.positional) == 0 {
		return ""
	}
	return strings.ToLower(g.positional[0])
}

func (g ghArgs) verb() string {
	if len(g.positional) < 2 {
		return ""
	}
	return strings.ToLower(g.positional[1])
}

// summary renders the command for the prompt, e.g. "gh pr comment".
func (g ghArgs) summary() string {
	parts := append([]string{"gh"}, g.positional...)
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, " ")
}

// localCommands never touch a remote repository: they read, or they only change
// local configuration.
var localCommands = map[string]bool{
	"alias":      true,
	"auth":       true,
	"browse":     true,
	"completion": true,
	"config":     true,
	"extension":  true,
	"help":       true,
	"search":     true,
	"status":     true,
	"version":    true,
}

// readVerbs lists, per gh command, the subcommands that only read. Anything not
// listed here counts as a write, so a gh subcommand added upstream after this
// table was written errs towards asking rather than towards silence.
var readVerbs = map[string]map[string]bool{
	"attestation": {"download": true, "verify": true},
	"cache":       {"list": true},
	"codespace":   {"list": true, "view": true},
	"gist":        {"clone": true, "list": true, "view": true},
	"issue":       {"list": true, "status": true, "view": true},
	"label":       {"list": true},
	"org":         {"list": true},
	"pr":          {"checkout": true, "checks": true, "diff": true, "list": true, "status": true, "view": true},
	"project":     {"field-list": true, "item-list": true, "list": true, "view": true},
	"release":     {"download": true, "list": true, "view": true},
	"repo":        {"clone": true, "list": true, "view": true},
	"ruleset":     {"check": true, "list": true, "view": true},
	"run":         {"download": true, "list": true, "view": true, "watch": true},
	"secret":      {"list": true},
	"variable":    {"get": true, "list": true},
	"workflow":    {"list": true, "view": true},
}

// isWrite reports whether the invocation changes remote state.
func (g ghArgs) isWrite() bool {
	sub := g.sub()
	switch {
	case sub == "":
		return false
	case localCommands[sub]:
		return false
	case sub == "api":
		// A method other than GET, or any field, means gh sends a mutation.
		return g.hasFields || (g.method != "" && !strings.EqualFold(g.method, "GET"))
	}
	if reads, known := readVerbs[sub]; known {
		return !reads[g.verb()]
	}
	return true
}

// hostFromRepo extracts the host from --repo, which gh accepts as
// [HOST/]OWNER/REPO or as a full URL.
func hostFromRepo(repo string) string {
	if repo == "" {
		return ""
	}
	if strings.Contains(repo, "://") {
		if u, err := url.Parse(repo); err == nil {
			return u.Hostname()
		}
		return ""
	}
	parts := strings.Split(repo, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		return parts[0]
	}
	return ""
}

// repoName reduces --repo to OWNER/REPO for the prompt.
func repoName(repo string) string {
	if repo == "" {
		return ""
	}
	if strings.Contains(repo, "://") {
		if u, err := url.Parse(repo); err == nil {
			repo = strings.Trim(u.Path, "/")
		}
	}
	repo = strings.TrimSuffix(repo, ".git")
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return repo
}

// hostFromGitRemote falls back to the origin remote of the working directory,
// which is how gh itself resolves the host when --repo is absent.
func hostFromGitRemote(cwd string) string {
	if cwd == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", cwd, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return hostFromGitURL(strings.TrimSpace(string(out)))
}

// hostFromGitURL handles both scp-like (git@host:owner/repo) and URL remotes.
func hostFromGitURL(remote string) string {
	if remote == "" {
		return ""
	}
	if !strings.Contains(remote, "://") {
		if at := strings.Index(remote, "@"); at >= 0 {
			remote = remote[at+1:]
		}
		if colon := strings.Index(remote, ":"); colon >= 0 {
			return remote[:colon]
		}
		return ""
	}
	u, err := url.Parse(remote)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, "/")
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}
