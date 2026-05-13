package typo

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Effect is one scheduled choreography: which cells to borrow, what path to
// follow, what triggers downstream effects. The Choreographer owns Effects
// while they're active.
type Effect struct {
	ID       string        // unique per scheduled effect; auto-assigned if empty
	Kind     string        // "scatter", "gather", "pluck"... drives EffectKind on CellTransform
	Path     PathFn        // motion logic
	Duration time.Duration // when this effect resolves
	Seed     int64         // per-event seed; drives PathFn variation
	Cells    []CellRef     // borrowed cells

	OriginX, OriginY float64 // path origin (typically 0 for relative motion)
	TargetX, TargetY float64 // path target (relative to origin)

	StartAt time.Time   // when to begin (zero = now)
	Reduced ReducedSpec // /quiet mode fallback

	Chains []ChainSpec // optional follow-ups
}

// ReducedSpec is the reduced-motion fallback for an Effect. Cells don't move;
// the effect's semantic state still applies (tint, brief pulse). Users with
// reduce_motion=true see this instead of Path-driven motion.
type ReducedSpec struct {
	// TintAlpha is a brief tint on the borrowed cells (e.g., for an award
	// conferral, the message glows warm). Duration = parent Effect.Duration.
	Tint        string // theme color name; empty = no tint
	PulseCells  bool   // if true, cells Pulse for parent Effect.Duration
}

// ChainSpec wires one Effect's end (or midpoint) to the start of another.
type ChainSpec struct {
	Next      *Effect
	Trigger   ChainTrigger
	Condition func() bool // optional gate; if returns false at trigger time, chain skips
	HandOff   bool        // if true, cells stay in transform across the A→B boundary
}

// ChainTrigger is when a chain fires relative to the parent effect's lifecycle.
type ChainTrigger int

const (
	OnComplete  ChainTrigger = iota // when parent ends
	OnMidpoint                       // at 50% of parent duration
	OnSettling                       // entering the last 25% of parent duration
)

// Choreographer coordinates Effect scheduling, chain firing, cell ownership,
// and reduced-motion fallback. One per Screen (or program-wide).
type Choreographer struct {
	mu           sync.RWMutex
	active       map[string]*runningEffect
	idCounter    int64
	reducedMode  bool
	now          func() time.Time
}

type runningEffect struct {
	effect      *Effect
	startedAt   time.Time
	chainsFired map[int]bool // indexed by Chains position
}

// NewChoreographer returns a configured Choreographer with system time.
func NewChoreographer() *Choreographer {
	return &Choreographer{
		active: map[string]*runningEffect{},
		now:    time.Now,
	}
}

// SetReducedMotion toggles the reduced-motion fallback. When on, scheduled
// Effects play their ReducedSpec instead of the Path-driven choreography.
func (c *Choreographer) SetReducedMotion(on bool) {
	c.mu.Lock()
	c.reducedMode = on
	c.mu.Unlock()
}

// ReducedMotion reports the current setting.
func (c *Choreographer) ReducedMotion() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.reducedMode
}

// Schedule queues an Effect to run. If e.StartAt is zero, runs immediately;
// otherwise starts at that time on the next Tick that crosses it. Returns
// the assigned effect ID for cancel/lookup.
func (c *Choreographer) Schedule(e *Effect) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e.ID == "" {
		n := atomic.AddInt64(&c.idCounter, 1)
		e.ID = "fx-" + strconv.FormatInt(n, 36)
	}
	start := e.StartAt
	if start.IsZero() {
		start = c.now()
	}
	c.active[e.ID] = &runningEffect{
		effect:      e,
		startedAt:   start,
		chainsFired: make(map[int]bool, len(e.Chains)),
	}
	return e.ID
}

// Cancel removes an active Effect by ID. Cells return to natural positions
// on the next Tick.
func (c *Choreographer) Cancel(id string) {
	c.mu.Lock()
	delete(c.active, id)
	c.mu.Unlock()
}

// Active reports how many Effects are currently running.
func (c *Choreographer) Active() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.active)
}

// Tick advances all active Effects, fires chain triggers, expires completed
// Effects, and returns the current CellTransform set for the renderer.
//
// In reduced-motion mode, returns no transforms (cells don't move) but
// chains still fire so persistent state changes still occur.
func (c *Choreographer) Tick(now time.Time) []CellTransform {
	c.mu.Lock()
	defer c.mu.Unlock()

	var transforms []CellTransform
	var toRemove []string

	for id, re := range c.active {
		// Effect hasn't started yet.
		if now.Before(re.startedAt) {
			continue
		}
		elapsed := now.Sub(re.startedAt)
		if elapsed >= re.effect.Duration {
			// Completed: fire OnComplete chains, mark for removal.
			c.fireChains(re, OnComplete, now)
			toRemove = append(toRemove, id)
			continue
		}

		// Fire mid-life chain triggers.
		if elapsed >= re.effect.Duration/2 {
			c.fireChains(re, OnMidpoint, now)
		}
		if elapsed >= (re.effect.Duration*3)/4 {
			c.fireChains(re, OnSettling, now)
		}

		// In reduced mode, no transforms — semantic state via ReducedSpec
		// applied by the renderer separately.
		if c.reducedMode {
			continue
		}

		// Emit a CellTransform per borrowed cell.
		for i, cellRef := range re.effect.Cells {
			args := PathArgs{
				Elapsed:   elapsed,
				Duration:  re.effect.Duration,
				CellIdx:   i,
				CellCount: len(re.effect.Cells),
				OriginX:   re.effect.OriginX,
				OriginY:   re.effect.OriginY,
				TargetX:   re.effect.TargetX,
				TargetY:   re.effect.TargetY,
				Seed:      re.effect.Seed,
			}
			dx, dy := re.effect.Path(args)
			transforms = append(transforms, CellTransform{
				SourceLayoutID: cellRef.LayoutID,
				SourceCellIdx:  cellRef.Idx,
				OffsetX:        dx,
				OffsetY:        dy,
				EffectKind:     re.effect.Kind,
			})
		}
	}

	for _, id := range toRemove {
		delete(c.active, id)
	}
	return transforms
}

// fireChains evaluates chain triggers that haven't fired yet on this Effect.
// Marks them fired so they only fire once each.
func (c *Choreographer) fireChains(re *runningEffect, trigger ChainTrigger, now time.Time) {
	for i, chain := range re.effect.Chains {
		if chain.Trigger != trigger {
			continue
		}
		if re.chainsFired[i] {
			continue
		}
		if chain.Condition != nil && !chain.Condition() {
			re.chainsFired[i] = true
			continue
		}
		// Schedule the next effect. HandOff is handled by reusing the source
		// effect's seed if requested, so cells don't visibly snap.
		if chain.Next != nil {
			next := *chain.Next // shallow copy so we don't mutate the spec
			if chain.HandOff && next.Seed == 0 {
				next.Seed = re.effect.Seed
			}
			if next.StartAt.IsZero() {
				next.StartAt = now
			}
			// Same locking convention: c.mu is already held by Tick.
			if next.ID == "" {
				n := atomic.AddInt64(&c.idCounter, 1)
				next.ID = "fx-" + strconv.FormatInt(n, 36)
			}
			c.active[next.ID] = &runningEffect{
				effect:      &next,
				startedAt:   next.StartAt,
				chainsFired: make(map[int]bool, len(next.Chains)),
			}
		}
		re.chainsFired[i] = true
	}
}

// IsTransformed reports whether a given (layoutID, cellIdx) is currently
// being rendered as a CellTransform — i.e., the renderer should skip drawing
// this cell at its natural position. Used by Render() to avoid double-render.
func (c *Choreographer) IsTransformed(layoutID string, cellIdx int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, re := range c.active {
		if c.reducedMode {
			return false
		}
		for _, cell := range re.effect.Cells {
			if cell.LayoutID == layoutID && cell.Idx == cellIdx {
				return true
			}
		}
	}
	return false
}
