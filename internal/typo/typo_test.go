package typo

import (
	"strings"
	"testing"
	"time"
)

// TestPrepareAddressesEveryChar covers the foundational contract: every
// glyph in the input text becomes an addressable Cell with a (col, row).
func TestPrepareAddressesEveryChar(t *testing.T) {
	l := Prepare("test-1", "hello world", 80)
	if l == nil {
		t.Fatal("Prepare returned nil")
	}
	// "hello world" = 11 chars
	if got := len(l.Cells); got != 11 {
		t.Errorf("len(Cells) = %d, want 11", got)
	}
	if l.Height != 1 {
		t.Errorf("Height = %d, want 1", l.Height)
	}
	if l.Width != 11 {
		t.Errorf("Width = %d, want 11", l.Width)
	}
	if l.Cells[0].Char != 'h' || l.Cells[10].Char != 'd' {
		t.Errorf("first/last chars wrong: %q / %q", l.Cells[0].Char, l.Cells[10].Char)
	}
}

// TestPrepareWrapsAtMaxWidth asserts that longer text gets wrapped, with
// later cells landing on subsequent rows.
func TestPrepareWrapsAtMaxWidth(t *testing.T) {
	text := "the quick brown fox jumps over the lazy dog"
	l := Prepare("wrap-test", text, 20)
	if l.Height < 2 {
		t.Errorf("Height = %d, expected wrap to produce at least 2 rows", l.Height)
	}
	// The last cell shouldn't be on row 0
	last := l.Cells[len(l.Cells)-1]
	if last.Row == 0 {
		t.Errorf("last cell still on row 0 — wrapping didn't happen: %+v", last)
	}
}

// TestTypeRevealProgresses asserts that Type drives Reveal from 0 to 1 over
// (perCharMs * cellCount) ms. This is what makes chat lines visibly arrive.
func TestTypeRevealProgresses(t *testing.T) {
	l := Prepare("t", "hello", 80) // 5 cells
	s := NewState()
	s.Reveal = 0

	// At t=0
	Type(&s, l, 0, 50)
	if s.Reveal != 0 {
		t.Errorf("at elapsed=0, Reveal=%v, want 0", s.Reveal)
	}

	// At t=halfway (125ms of 250ms total)
	Type(&s, l, 125*time.Millisecond, 50)
	if s.Reveal < 0.4 || s.Reveal > 0.6 {
		t.Errorf("at half, Reveal=%v, want ~0.5", s.Reveal)
	}

	// At t=done
	Type(&s, l, 300*time.Millisecond, 50)
	if s.Reveal != 1.0 {
		t.Errorf("at done, Reveal=%v, want 1.0", s.Reveal)
	}
}

// TestRenderRespectsReveal asserts that with Reveal=0.5, roughly half the
// chars render. The visible chars should be from the LEFT (Type direction).
func TestRenderRespectsReveal(t *testing.T) {
	l := Prepare("r", "abcdefghij", 80) // 10 cells
	s := NewState()
	s.Reveal = 0.5
	rows := Render(l, &s, time.Now())
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Strip ANSI then count visible chars
	visible := stripANSI(rows[0])
	// Trim trailing spaces from padding
	visible = strings.TrimRight(visible, " ")
	if got := len(visible); got != 5 {
		t.Errorf("at Reveal=0.5, visible chars=%d, want 5: %q", got, visible)
	}
	if !strings.HasPrefix(visible, "abcde") {
		t.Errorf("visible should start with 'abcde', got %q", visible)
	}
}

// TestGreetMacroFinishesByTotal asserts the Greet macro returns done=true
// once the per-char window has elapsed.
func TestGreetMacroFinishesByTotal(t *testing.T) {
	l := Prepare("g", "abc", 80) // 3 cells
	s := NewState()
	s.Reveal = 0
	macro := Greet()

	// Not yet done at 0ms
	if done := macro(&s, l, 0, time.Now()); done {
		t.Error("Greet should not be done at elapsed=0")
	}

	// Done after 3*30ms = 90ms (plus a buffer)
	if done := macro(&s, l, 200*time.Millisecond, time.Now()); !done {
		t.Error("Greet should be done after 200ms for 3 chars at 30ms/char")
	}
}

func stripANSI(s string) string {
	// Trivial strip — sufficient for tests
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
