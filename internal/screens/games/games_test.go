package games

import (
	"strings"
	"testing"
)

// TestRenderListHasGames asserts that the launcher actually renders the
// games when the screen is in list state. Caught here so the user-reported
// "/games does nothing" stays diagnosed: if this passes, the games view is
// fine and any blank-screen report is upstream of this package.
func TestRenderListHasGames(t *testing.T) {
	s := New(nil)
	out := s.View(120, 30)
	if out == "" {
		t.Fatal("View returned empty string")
	}
	if !strings.Contains(out, "bricks blitz") {
		t.Errorf("games launcher missing 'bricks blitz' entry; got:\n%s", out)
	}
	if !strings.Contains(out, "mini-distractions") {
		t.Errorf("games launcher missing title line; got:\n%s", out)
	}
}
