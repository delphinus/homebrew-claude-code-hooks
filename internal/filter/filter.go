package filter

import (
	"regexp"
	"strings"
)

var skipCommands = map[string]bool{
	"ls": true, "cat": true, "head": true, "tail": true, "wc": true,
	"file": true, "stat": true, "which": true, "where": true, "type": true,
	"echo": true, "printf": true, "pwd": true, "cd": true, "test": true,
	"true": true, "false": true, "grep": true, "rg": true, "find": true,
	"diff": true, "sort": true, "uniq": true, "tr": true, "cut": true,
	"mkdir": true, "rmdir": true, "rm": true, "cp": true, "mv": true,
	"ln": true, "chmod": true, "chown": true, "touch": true,
	"basename": true, "dirname": true, "realpath": true, "readlink": true,
	"tree": true, "du": true, "df": true, "less": true, "more": true,
	"xargs": true, "tee": true, "whoami": true, "hostname": true,
	"date": true, "uname": true, "env": true, "set": true, "export": true,
	"alias": true, "id": true, "jq": true,
}

// git: read-only subcommands to skip
var gitSkipSubcommands = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true,
	"rev-parse": true, "describe": true, "shortlog": true,
	"ls-files": true, "ls-tree": true, "cat-file": true,
	"reflog": true, "blame": true,
}

// go: build/test/inspection subcommands to skip
var goSkipSubcommands = map[string]bool{
	"test": true, "build": true, "vet": true,
	"version": true, "env": true, "list": true, "doc": true,
}

// gh: read-only verbs (3rd token) to skip
var ghSkipVerbs = map[string]bool{
	"list": true, "view": true, "watch": true,
	"status": true, "checks": true, "diff": true,
}

var splitPattern = regexp.MustCompile(`[|;&]+`)

// isSkippedSegment returns true if a single command segment should be skipped.
func isSkippedSegment(seg string) bool {
	tokens := strings.Fields(seg)
	if len(tokens) == 0 {
		return true
	}
	firstToken := tokens[0]

	if skipCommands[firstToken] {
		return true
	}

	switch firstToken {
	case "git":
		return len(tokens) >= 2 && gitSkipSubcommands[tokens[1]]
	case "go":
		return len(tokens) >= 2 && goSkipSubcommands[tokens[1]]
	case "gh":
		return len(tokens) >= 3 && ghSkipVerbs[tokens[2]]
	}

	return false
}

// ShouldRecordCommand returns true if the command contains at least one
// non-blocklisted command. Pipe, semicolon, &&, and || segments are checked
// individually.
func ShouldRecordCommand(cmd string) bool {
	segments := splitPattern.Split(cmd, -1)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if !isSkippedSegment(seg) {
			return true
		}
	}
	return false
}
