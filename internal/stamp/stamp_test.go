package stamp

import "testing"

func TestRunNoopOutsideKitty(t *testing.T) {
	// Without KITTY_WINDOW_ID there is no window to stamp. This must not invoke
	// kitten: the test suite usually runs inside a real kitty window, and a
	// stray set-user-vars would land on it.
	t.Setenv("KITTY_WINDOW_ID", "")
	if err := Run("11111111-2222-3333-4444-555555555555"); err != nil {
		t.Errorf("Run outside kitty = %v, want nil", err)
	}
}

func TestRunNoopWithoutSessionID(t *testing.T) {
	// SessionStart always carries a session id, but a malformed payload must not
	// clear the var that a previous, correct stamp wrote.
	t.Setenv("KITTY_WINDOW_ID", "1")
	if err := Run(""); err != nil {
		t.Errorf("Run with empty session id = %v, want nil", err)
	}
}

func TestClearNoopOutsideKitty(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "")
	if err := Clear(); err != nil {
		t.Errorf("Clear outside kitty = %v, want nil", err)
	}
}
