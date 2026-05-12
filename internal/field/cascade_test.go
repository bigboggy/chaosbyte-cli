package field

import (
	"testing"
	"time"
)

// TestCascadeExpires asserts that a cascade is dropped after its Decay
// window passes on the next Tick. This is the behavioral guarantee Daniel
// asked for: cascades are events, not ambient state, so they have to
// reliably go away.
func TestCascadeExpires(t *testing.T) {
	e := NewEngine()
	e.Resize(40, 8)
	now := time.Now()
	e.AddCascade(CascadeLine{
		Row:    0,
		Text:   "join @alice",
		BornAt: now,
		Decay:  100 * time.Millisecond,
	})
	if len(e.cascades) != 1 {
		t.Fatalf("after AddCascade, len(cascades) = %d, want 1", len(e.cascades))
	}
	if len(e.fgLines) != 1 {
		t.Fatalf("after AddCascade, len(fgLines) = %d, want 1", len(e.fgLines))
	}

	// Advance past the decay window.
	e.Tick(now.Add(250 * time.Millisecond))
	if len(e.cascades) != 0 {
		t.Fatalf("after expiry tick, len(cascades) = %d, want 0", len(e.cascades))
	}
	if len(e.fgLines) != 0 {
		t.Fatalf("after expiry tick, fgLines should be empty; got %d", len(e.fgLines))
	}
}

// TestCascadeReplacesSameRow asserts that adding a cascade on a row that
// already has one swaps the text rather than stacking.
func TestCascadeReplacesSameRow(t *testing.T) {
	e := NewEngine()
	e.Resize(40, 8)
	e.AddCascade(CascadeLine{Row: 0, Text: "first", Decay: time.Second})
	e.AddCascade(CascadeLine{Row: 0, Text: "second", Decay: time.Second})
	if len(e.cascades) != 1 {
		t.Errorf("len(cascades) = %d, want 1 after replacement", len(e.cascades))
	}
	if e.cascades[0].Text != "second" {
		t.Errorf("cascades[0].Text = %q, want %q", e.cascades[0].Text, "second")
	}
}
