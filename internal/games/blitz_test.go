package games

import (
	"testing"
	"time"
)

// TestNewBlitzPicksTarget confirms a fresh round always has a non-empty
// target word selected from the curated bank. The lobby uses Target()
// right after construction to fire the cascading announcement; an empty
// target would leave the announcement bare.
func TestNewBlitzPicksTarget(t *testing.T) {
	b := NewBlitz(time.Unix(0, 0))
	if b.Target() == "" {
		t.Fatal("NewBlitz must pick a target word from the bank")
	}
}

// TestMatchScoreFirstThree confirms the 3 / 2 / 1 scoring ladder lands
// on the first three unique authors in match order. The fourth and
// later matches still record but score 1.
func TestMatchScoreFirstThree(t *testing.T) {
	b := NewBlitz(time.Unix(0, 0))
	target := b.Target()
	body := "i'll " + target + " it"

	pts, ok := b.MatchScore("@alice", body)
	if !ok || pts != 3 {
		t.Fatalf("first match should score 3, got pts=%d ok=%v", pts, ok)
	}
	pts, ok = b.MatchScore("@bob", body)
	if !ok || pts != 2 {
		t.Fatalf("second match should score 2, got pts=%d ok=%v", pts, ok)
	}
	pts, ok = b.MatchScore("@cleo", body)
	if !ok || pts != 1 {
		t.Fatalf("third match should score 1, got pts=%d ok=%v", pts, ok)
	}
	pts, ok = b.MatchScore("@dani", body)
	if !ok || pts != 1 {
		t.Fatalf("fourth match should still record at 1, got pts=%d ok=%v", pts, ok)
	}
}

// TestMatchScoreRejectsRepeats confirms a player who has already scored
// cannot score again on the same target. Spam-typing the target word
// can't keep ratcheting your score up.
func TestMatchScoreRejectsRepeats(t *testing.T) {
	b := NewBlitz(time.Unix(0, 0))
	body := "shipping " + b.Target() + " today"

	if _, ok := b.MatchScore("@alice", body); !ok {
		t.Fatal("first match should record")
	}
	if pts, ok := b.MatchScore("@alice", body); ok || pts != 0 {
		t.Fatalf("repeat match should be rejected, got pts=%d ok=%v", pts, ok)
	}
}

// TestMatchScoreRequiresWholeToken confirms partial matches don't count.
// If the target is "ship", typing "shipping" should not score because
// the substring isn't the whole token.
func TestMatchScoreRequiresWholeToken(t *testing.T) {
	b := NewBlitz(time.Unix(0, 0))
	target := b.Target()
	// Build a body that contains target as a substring of a longer word.
	// "ship" -> "shipping"; "rust" -> "rusting"; etc. Append a generic
	// suffix that doesn't collide with the bank.
	body := target + "zzz only"

	if pts, ok := b.MatchScore("@alice", body); ok || pts != 0 {
		t.Fatalf("substring-only match should be rejected, got pts=%d ok=%v", pts, ok)
	}
}

// TestTickResolvesWinnerAtOffsetEntry confirms the winner is set at the
// boundary between main and offset, so the lobby has the full offset
// window to play the winner cascade through.
func TestTickResolvesWinnerAtOffsetEntry(t *testing.T) {
	start := time.Unix(0, 0)
	b := NewBlitz(start)
	body := "let's " + b.Target() + " it"
	b.MatchScore("@alice", body)

	b.Tick(start.Add(DefaultDuration + 100*time.Millisecond))

	winner, ready := b.WinnerReady()
	if !ready {
		t.Fatal("WinnerReady should fire once duration is crossed")
	}
	if winner != "@alice" {
		t.Fatalf("winner = %q, want @alice", winner)
	}
	if b.Done() {
		t.Fatal("Done should not yet be true at offset entry")
	}
}

// TestDoneFlipsAfterOffset confirms Done waits for the full
// duration + offsetDuration before flipping. The lobby holds the
// *Blitz reference across the offset window so the cascade can play.
func TestDoneFlipsAfterOffset(t *testing.T) {
	start := time.Unix(0, 0)
	b := NewBlitz(start)
	b.Tick(start.Add(DefaultDuration + offsetDuration + time.Second))
	if !b.Done() {
		t.Fatal("Done should flip past duration + offset")
	}
}

// TestWinnerFallsBackToRoom confirms an empty round names "the room"
// so the closing cascade has a target.
func TestWinnerFallsBackToRoom(t *testing.T) {
	start := time.Unix(0, 0)
	b := NewBlitz(start)
	b.Tick(start.Add(DefaultDuration + offsetDuration + time.Second))
	if w := b.Winner(); w != "the room" {
		t.Fatalf("empty round should resolve to \"the room\", got %q", w)
	}
}

// TestStandingsSortedHighToLow confirms the leaderboard returns scores
// in descending order. The lobby renders this in the top bar so the
// leader sits at the left.
func TestStandingsSortedHighToLow(t *testing.T) {
	b := NewBlitz(time.Unix(0, 0))
	body := "we " + b.Target() + " today"

	b.MatchScore("@alice", body)
	b.MatchScore("@bob", body)
	b.MatchScore("@cleo", body)

	standings := b.Standings()
	if len(standings) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(standings))
	}
	if standings[0].Author != "@alice" || standings[0].Points != 3 {
		t.Fatalf("first standing should be alice with 3 points, got %+v", standings[0])
	}
	if standings[2].Author != "@cleo" || standings[2].Points != 1 {
		t.Fatalf("last standing should be cleo with 1 point, got %+v", standings[2])
	}
}
