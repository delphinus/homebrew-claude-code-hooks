package filter

import "testing"

func TestShouldRecordCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		// basic skip commands
		{"ls", false},
		{"ls -la", false},
		{"cat foo.txt", false},
		{"grep pattern file | sort | uniq", false},
		{"npm install", true},
		{"cat foo && npm test", true},
		{"echo hello; python script.py", true},
		{"", false},
		{"docker compose up", true},
		{"mkdir -p /tmp/test && cd /tmp/test", false},
		{"jq .foo bar.json", false},

		// git read-only → skip
		{"git status", false},
		{"git diff --cached", false},
		{"git log --oneline -5", false},
		{"git show HEAD", false},
		{"git rev-parse v2.3.0", false},
		{"git blame file.go", false},

		// git state-changing → record
		{"git commit -m 'msg'", true},
		{"git push origin main", true},
		{"git tag v1.0.0", true},
		{"git pull --rebase origin main", true},
		{"git add file.go", true},
		{"git merge feature", true},

		// go build/test/inspection → skip
		{"go test ./...", false},
		{"go build ./...", false},
		{"go vet ./...", false},

		// go state-changing → record
		{"go run main.go", true},
		{"go install ...", true},
		{"go get ...", true},
		{"go mod tidy", true},

		// gh read-only → skip
		{"gh run list --limit 5", false},
		{"gh run view 123 --log", false},
		{"gh run watch 123", false},
		{"gh pr list", false},
		{"gh pr view 123", false},
		{"gh pr checks 123", false},
		{"gh issue list", false},

		// gh state-changing → record
		{"gh pr create --title 'title'", true},
		{"gh pr close 123", true},
		{"gh pr merge 123", true},
		{"gh release create v1.0.0", true},
		{"gh release delete v1.0.0 --yes", true},
		{"gh run cancel 123", true},

		// pipe: all segments skipped → skip
		{"git log --oneline | head -5", false},
		{"git status && git diff", false},

		// pipe: at least one recorded → record
		{"git diff && git commit -m 'msg'", true},
		{"rm -rf dist && go build", false},
		{"ls | go build", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := ShouldRecordCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("ShouldRecordCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
