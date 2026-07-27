package notify

import (
	"reflect"
	"testing"
)

func TestHelperArgs(t *testing.T) {
	got := helperArgs("Claude Code", "作業が完了しました", "10")
	want := []string{
		"post",
		"--title", "Claude Code",
		"--message", "作業が完了しました",
		"--pane", "10",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("helperArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
