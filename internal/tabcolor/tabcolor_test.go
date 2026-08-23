package tabcolor

import "testing"

func TestRunUnknownState(t *testing.T) {
	if err := Run("bogus"); err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestRunNoopOutsideSupportedTerminals(t *testing.T) {
	// With neither env var set, Run must not invoke kitten or wezterm and must
	// succeed. Both have to be cleared: the test suite itself usually runs inside
	// one of these terminals, and without this it would repaint a real tab.
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("WEZTERM_PANE", "")
	for state := range validStates {
		if err := Run(state); err != nil {
			t.Errorf("Run(%q) = %v, want nil", state, err)
		}
	}
}

func TestStateColorsCoverEveryColoredState(t *testing.T) {
	for state := range validStates {
		if state == "default" {
			// "default" clears the colors rather than setting them.
			if _, ok := stateColors[state]; ok {
				t.Errorf("stateColors must not define %q", state)
			}
			continue
		}
		c, ok := stateColors[state]
		if !ok {
			t.Errorf("stateColors is missing %q", state)
			continue
		}
		if len(c.activeBG) != 7 || c.activeBG[0] != '#' {
			t.Errorf("%q: bad activeBG %q", state, c.activeBG)
		}
		if len(c.inactiveBG) != 7 || c.inactiveBG[0] != '#' {
			t.Errorf("%q: bad inactiveBG %q", state, c.inactiveBG)
		}
	}
}
