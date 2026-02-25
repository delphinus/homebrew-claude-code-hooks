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

var splitPattern = regexp.MustCompile(`[|;&]+`)

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
		// Extract the first token (the command name)
		firstToken := seg
		if idx := strings.IndexAny(seg, " \t"); idx >= 0 {
			firstToken = seg[:idx]
		}
		if !skipCommands[firstToken] {
			return true
		}
	}
	return false
}
