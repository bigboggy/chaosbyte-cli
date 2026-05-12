package typo

import (
	"math"
	"math/rand"
	"time"
)

// CellTransform represents a cell that has been temporarily "borrowed" from
// its natural Layout position for an event. While a transform is active,
// the cell renders at its PathFn-computed position instead of (Col, Row).
//
// Transforms have lifespans. When BornAt + Duration passes, the cell returns
// to its natural position on the next Tick. Multiple transforms can target
// the same cell only via the choreographer's merge logic; the renderer
// itself trusts that the active set is collision-free.
type CellTransform struct {
	SourceLayoutID string // which Layout this cell belongs to
	SourceCellIdx  int    // index into that Layout's Cells slice
	Char           rune   // the actual rune to render (may differ from source if transforming)

	Path     PathFn
	Duration time.Duration
	BornAt   time.Time
	Seed     int64 // per-event seed, drives PathFn determinism + variation
	TargetX  float64
	TargetY  float64
	OriginX  float64 // optional anchor different from source cell's natural position
	OriginY  float64
}

// PathArgs is the input to every PathFn. Frozen on call so PathFns are pure.
type PathArgs struct {
	Elapsed   time.Duration
	Duration  time.Duration
	CellIdx   int
	CellCount int
	OriginX   float64
	OriginY   float64
	TargetX   float64
	TargetY   float64
	Seed      int64
}

// PathFn computes the (x, y) screen offset for one cell at a given moment.
// All PathFns must be pure: same args → same output. The Seed parameter is
// what enables predictable variation — same effect kind across two firings
// uses the same PathFn but different seeds, producing recognizably similar
// but never-identical choreographies.
type PathFn func(args PathArgs) (offsetX, offsetY float64)

// Progress returns elapsed/duration clamped to [0, 1].
func (a PathArgs) Progress() float64 {
	if a.Duration <= 0 {
		return 1
	}
	t := float64(a.Elapsed) / float64(a.Duration)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// rng returns a deterministic rng for this cell within this event. Combining
// the event seed with the cell index ensures each cell varies independently
// while the whole event stays reproducible from the seed.
func (a PathArgs) rng() *rand.Rand {
	return rand.New(rand.NewSource(a.Seed ^ int64(a.CellIdx+1)*2654435761))
}

// jitter returns a small per-cell jitter value in [-amount, amount].
func (a PathArgs) jitter(amount float64) float64 {
	return (a.rng().Float64()*2 - 1) * amount
}

// easeOutCubic decelerates toward 1.
func easeOutCubic(t float64) float64 {
	t = 1 - t
	return 1 - t*t*t
}

// easeInOutCubic accelerates then decelerates.
func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	p := 2*t - 2
	return 1 + p*p*p/2
}

// Scatter sends cells outward from origin, hold at radius, return. The radial
// distribution is jittered per cell so cells don't end up uniformly spaced.
// Signature: starburst that returns.
func Scatter(radius float64) PathFn {
	return func(args PathArgs) (float64, float64) {
		t := args.Progress()
		// Three phases: outward (0..0.4), hold (0.4..0.6), return (0.6..1)
		var amt float64
		switch {
		case t < 0.4:
			amt = easeOutCubic(t / 0.4)
		case t < 0.6:
			amt = 1.0
		default:
			amt = 1 - easeInOutCubic((t-0.6)/0.4)
		}
		// Angle: distribute around circle by cell index + jitter
		baseAngle := 2 * math.Pi * float64(args.CellIdx) / float64(maxInt(args.CellCount, 1))
		angle := baseAngle + args.jitter(0.3)
		r := radius * (0.7 + args.rng().Float64()*0.3) // per-cell radius variation
		return amt * r * math.Cos(angle), amt * r * math.Sin(angle)
	}
}

// Orbit traces a circle around an origin point. cells offset from each other
// so they trail rather than overlap.
func Orbit(radius float64, revolutions float64) PathFn {
	return func(args PathArgs) (float64, float64) {
		t := args.Progress()
		// Each cell is offset on the circle by its index
		phase := float64(args.CellIdx) * 2 * math.Pi / float64(maxInt(args.CellCount, 1))
		angle := phase + revolutions*2*math.Pi*t
		// Easing: fade in and out at the edges so we don't snap into orbit
		envelope := math.Sin(t * math.Pi)
		r := radius * envelope
		return r * math.Cos(angle), r * math.Sin(angle)
	}
}

// Pluck moves a cell from its natural position to a target position. The
// path is a quadratic curve through a control point above the midpoint so
// the cell "lifts" rather than slides linearly.
func Pluck() PathFn {
	return func(args PathArgs) (float64, float64) {
		t := easeInOutCubic(args.Progress())
		// Lift the trajectory: control point is the midpoint shifted upward
		ctrlX := (args.OriginX + args.TargetX) / 2
		ctrlY := (args.OriginY+args.TargetY)/2 - 2.0 // 2 cells up
		// Quadratic bezier
		x := (1-t)*(1-t)*args.OriginX + 2*(1-t)*t*ctrlX + t*t*args.TargetX
		y := (1-t)*(1-t)*args.OriginY + 2*(1-t)*t*ctrlY + t*t*args.TargetY
		// Per-cell variation: small angular jitter on the path
		jx := args.jitter(0.4) * math.Sin(t*math.Pi)
		jy := args.jitter(0.2) * math.Sin(t*math.Pi)
		return x - args.OriginX + jx, y - args.OriginY + jy
	}
}

// Gather converges multiple cells onto a target point, holds briefly, then
// returns each cell to its own origin. Used for award conferral, mod summons.
// Per-seed variation comes through: per-cell stagger (timing) and a small
// orthogonal arc offset during the gather phase (curve shape).
func Gather() PathFn {
	return func(args PathArgs) (float64, float64) {
		t := args.Progress()
		// Three phases: gather (0..0.4), hold (0.4..0.65), release (0.65..1)
		var phase float64
		switch {
		case t < 0.4:
			phase = easeOutCubic(t / 0.4)
		case t < 0.65:
			phase = 1.0
		default:
			phase = 1 - easeInOutCubic((t-0.65)/0.35)
		}
		dx := args.TargetX - args.OriginX
		dy := args.TargetY - args.OriginY
		// Per-cell timing stagger seeded by event seed so each event has its
		// own "rhythm" of arrival.
		stagger := args.jitter(0.04)
		adjusted := phase - stagger
		if adjusted < 0 {
			adjusted = 0
		}
		// Orthogonal arc: cells curve in instead of going straight. Sign
		// and magnitude vary per cell + seed, so each gather has a
		// different "swirl" pattern.
		arcMag := args.jitter(1.2)
		// Perpendicular direction to (dx, dy)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 {
			dist = 1
		}
		px := -dy / dist
		py := dx / dist
		arcEnvelope := math.Sin(adjusted * math.Pi) // peaks mid-flight
		// Hold-phase micro-jitter so converged cells don't sit perfectly still
		if t > 0.4 && t < 0.65 {
			hold := (t - 0.4) / 0.25
			adjusted = 1 + args.jitter(0.15)*math.Sin(hold*8*math.Pi)
			arcEnvelope = 0
		}
		return adjusted*dx + arcMag*arcEnvelope*px,
			adjusted*dy + arcMag*arcEnvelope*py
	}
}

// Vibrate jitters cells in place. Low amplitude per frame, settles to 0 at end.
func Vibrate(amplitude float64) PathFn {
	return func(args PathArgs) (float64, float64) {
		t := args.Progress()
		envelope := 1 - t // amplitude decays linearly
		// Use a time-bucketed rng so cells re-jitter every ~50ms instead of every frame
		bucket := int64(args.Elapsed / (50 * time.Millisecond))
		r := rand.New(rand.NewSource(args.Seed ^ int64(args.CellIdx) ^ bucket))
		return (r.Float64()*2 - 1) * amplitude * envelope,
			(r.Float64()*2 - 1) * amplitude * envelope * 0.5 // less vertical
	}
}

// Ripple expands outward from origin: cells closer to origin transform first,
// cells farther transform later. Used for room-wide responses to a single
// event (mass reaction wave).
func Ripple(speed float64) PathFn {
	return func(args PathArgs) (float64, float64) {
		t := args.Progress()
		// Distance of this cell from origin
		dx := args.OriginX - args.TargetX
		dy := args.OriginY - args.TargetY
		dist := math.Sqrt(dx*dx + dy*dy)
		// Wavefront position at this time
		front := speed * float64(args.Elapsed) / float64(time.Second)
		// Cell is in the wave when wave passes its distance
		wavePos := front - dist
		if wavePos < 0 || wavePos > 1.5 {
			return 0, 0
		}
		// Cell "lifts" briefly as the wave passes
		envelope := math.Sin(wavePos * math.Pi / 1.5)
		// Direction: away from origin
		_ = t // unused for now; intensity ramps via wave envelope
		if dist < 0.001 {
			return 0, -envelope * 0.5 // origin cell lifts up
		}
		return -envelope * 0.3 * dx / dist, -envelope * 0.3 * dy / dist
	}
}

// Drift moves cells in a steady direction. Optional decay fades it out at
// the end. Used for floating reactions, repo names rising into trending.
func Drift(angleRadians float64, distance float64) PathFn {
	return func(args PathArgs) (float64, float64) {
		t := easeOutCubic(args.Progress())
		dx := math.Cos(angleRadians) * distance * t
		dy := math.Sin(angleRadians) * distance * t
		// Per-cell small drift variation
		dx += args.jitter(0.15) * t
		dy += args.jitter(0.15) * t
		return dx, dy
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
