package tabcolor

import "testing"

func TestRunUnknownState(t *testing.T) {
	if err := Run("bogus"); err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestRunNoopOutsideWezTerm(t *testing.T) {
	// With WEZTERM_PANE unset, Run must not invoke wezterm and must succeed.
	t.Setenv("WEZTERM_PANE", "")
	for state := range validStates {
		if err := Run(state); err != nil {
			t.Errorf("Run(%q) = %v, want nil", state, err)
		}
	}
}
