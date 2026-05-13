package typo

import (
	"testing"
	"time"
)

// TestSchedulePopulatesActive asserts that Schedule registers an Effect so
// Tick can find it. Foundational; nothing else works if this breaks.
func TestSchedulePopulatesActive(t *testing.T) {
	c := NewChoreographer()
	id := c.Schedule(&Effect{
		Kind:     "pluck",
		Path:     Pluck(),
		Duration: 500 * time.Millisecond,
		Cells:    []CellRef{{LayoutID: "msg-1", Idx: 0}},
	})
	if id == "" {
		t.Fatal("Schedule returned empty ID")
	}
	if c.Active() != 1 {
		t.Errorf("Active() = %d, want 1", c.Active())
	}
}

// TestTickReturnsTransformsForActiveEffects asserts that an active Effect
// produces a CellTransform per borrowed cell on Tick.
func TestTickReturnsTransformsForActiveEffects(t *testing.T) {
	c := NewChoreographer()
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "gather",
		Path:     Gather(),
		Duration: 1 * time.Second,
		Seed:     42,
		Cells: []CellRef{
			{LayoutID: "msg-1", Idx: 0},
			{LayoutID: "msg-1", Idx: 1},
			{LayoutID: "msg-2", Idx: 0},
		},
		TargetX: 10, TargetY: 0,
		StartAt: start,
	})
	transforms := c.Tick(start.Add(200 * time.Millisecond))
	if len(transforms) != 3 {
		t.Errorf("expected 3 transforms (one per cell), got %d", len(transforms))
	}
	for _, tr := range transforms {
		if tr.EffectKind != "gather" {
			t.Errorf("EffectKind = %q, want gather", tr.EffectKind)
		}
	}
}

// TestEffectExpires asserts that an Effect past its Duration is dropped.
func TestEffectExpires(t *testing.T) {
	c := NewChoreographer()
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "vibrate",
		Path:     Vibrate(0.5),
		Duration: 500 * time.Millisecond,
		Cells:    []CellRef{{LayoutID: "x", Idx: 0}},
		StartAt:  start,
	})
	if c.Active() != 1 {
		t.Fatalf("pre-expiry Active = %d", c.Active())
	}
	c.Tick(start.Add(700 * time.Millisecond))
	if c.Active() != 0 {
		t.Errorf("post-expiry Active = %d, want 0", c.Active())
	}
}

// TestOnCompleteChainFires asserts a chained Effect starts when the parent
// reaches its Duration.
func TestOnCompleteChainFires(t *testing.T) {
	c := NewChoreographer()
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "parent",
		Path:     Pluck(),
		Duration: 300 * time.Millisecond,
		Cells:    []CellRef{{LayoutID: "a", Idx: 0}},
		StartAt:  start,
		Chains: []ChainSpec{
			{
				Trigger: OnComplete,
				Next: &Effect{
					Kind:     "child",
					Path:     Gather(),
					Duration: 200 * time.Millisecond,
					Cells:    []CellRef{{LayoutID: "b", Idx: 0}},
				},
			},
		},
	})
	// Before parent expires: 1 active.
	if c.Active() != 1 {
		t.Fatalf("pre-complete Active = %d", c.Active())
	}
	c.Tick(start.Add(350 * time.Millisecond))
	// Parent expired, child started.
	if c.Active() != 1 {
		t.Errorf("post-complete Active = %d, want 1 (the child)", c.Active())
	}
	transforms := c.Tick(start.Add(400 * time.Millisecond))
	foundChild := false
	for _, tr := range transforms {
		if tr.EffectKind == "child" {
			foundChild = true
			break
		}
	}
	if !foundChild {
		t.Error("child effect didn't produce transforms after parent completed")
	}
}

// TestChainConditionGate asserts that a chain with a false-returning
// Condition skips its Next effect.
func TestChainConditionGate(t *testing.T) {
	c := NewChoreographer()
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "parent",
		Path:     Pluck(),
		Duration: 200 * time.Millisecond,
		Cells:    []CellRef{{LayoutID: "a", Idx: 0}},
		StartAt:  start,
		Chains: []ChainSpec{
			{
				Trigger:   OnComplete,
				Condition: func() bool { return false }, // gate closed
				Next: &Effect{
					Kind:     "child",
					Path:     Pluck(),
					Duration: 200 * time.Millisecond,
					Cells:    []CellRef{{LayoutID: "b", Idx: 0}},
				},
			},
		},
	})
	c.Tick(start.Add(300 * time.Millisecond))
	if c.Active() != 0 {
		t.Errorf("gate-closed Active = %d, want 0 (parent expired, child gated)", c.Active())
	}
}

// TestReducedMotionSuppresses asserts that in reduced-motion mode, Tick
// returns no transforms. Chain firing should still work for persistent state.
func TestReducedMotionSuppresses(t *testing.T) {
	c := NewChoreographer()
	c.SetReducedMotion(true)
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "gather",
		Path:     Gather(),
		Duration: 1 * time.Second,
		Cells:    []CellRef{{LayoutID: "x", Idx: 0}},
		StartAt:  start,
	})
	transforms := c.Tick(start.Add(200 * time.Millisecond))
	if len(transforms) != 0 {
		t.Errorf("reduced-motion produced %d transforms; want 0", len(transforms))
	}
}

// TestIsTransformedDetectsBorrowed asserts the cell-ownership check works
// so the renderer can skip cells currently in transform.
func TestIsTransformedDetectsBorrowed(t *testing.T) {
	c := NewChoreographer()
	c.Schedule(&Effect{
		Kind:     "scatter",
		Path:     Scatter(3),
		Duration: 1 * time.Second,
		Cells:    []CellRef{{LayoutID: "msg-1", Idx: 5}},
	})
	if !c.IsTransformed("msg-1", 5) {
		t.Error("IsTransformed missed a known-borrowed cell")
	}
	if c.IsTransformed("msg-1", 99) {
		t.Error("IsTransformed false-positive on a non-borrowed cell")
	}
	if c.IsTransformed("other-msg", 5) {
		t.Error("IsTransformed false-positive on a different layout")
	}
}

// TestChainHandOffPreservesSeed asserts that a chained Effect with HandOff=true
// inherits the parent's seed, so variation stays coherent across the boundary.
func TestChainHandOffPreservesSeed(t *testing.T) {
	c := NewChoreographer()
	start := time.Now()
	c.Schedule(&Effect{
		Kind:     "parent",
		Path:     Pluck(),
		Duration: 100 * time.Millisecond,
		Cells:    []CellRef{{LayoutID: "a", Idx: 0}},
		Seed:     12345,
		StartAt:  start,
		Chains: []ChainSpec{
			{
				Trigger: OnComplete,
				HandOff: true,
				Next: &Effect{
					Kind:     "child",
					Path:     Gather(),
					Duration: 100 * time.Millisecond,
					Cells:    []CellRef{{LayoutID: "a", Idx: 0}},
					// Seed deliberately zero so HandOff fills it
				},
			},
		},
	})
	c.Tick(start.Add(150 * time.Millisecond))
	// Active = 1 child. Find it.
	c.tickInternal(func(re *runningEffect) {
		if re.effect.Kind == "child" {
			if re.effect.Seed != 12345 {
				t.Errorf("HandOff didn't preserve seed: got %d, want 12345", re.effect.Seed)
			}
		}
	})
}

// tickInternal is a test helper. Walks active effects.
func (c *Choreographer) tickInternal(visit func(*runningEffect)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, re := range c.active {
		visit(re)
	}
}
