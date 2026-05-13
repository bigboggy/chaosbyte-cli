package typo

import (
	"math"
	"testing"
	"time"
)

// TestPathFnsDeterministic asserts that calling any PathFn twice with the
// same args returns the same offset. This is the contract — without it,
// renders would flicker frame-to-frame on cells whose state hasn't changed.
func TestPathFnsDeterministic(t *testing.T) {
	args := PathArgs{
		Elapsed: 200 * time.Millisecond,
		Duration: 800 * time.Millisecond,
		CellIdx:   3,
		CellCount: 10,
		OriginX:   0,
		OriginY:   0,
		TargetX:   20,
		TargetY:   5,
		Seed:      42,
	}
	fns := map[string]PathFn{
		"Scatter": Scatter(4),
		"Orbit":   Orbit(3, 1),
		"Pluck":   Pluck(),
		"Gather":  Gather(),
		"Vibrate": Vibrate(0.5),
		"Ripple":  Ripple(20),
		"Drift":   Drift(math.Pi/2, 5),
	}
	for name, fn := range fns {
		x1, y1 := fn(args)
		x2, y2 := fn(args)
		if x1 != x2 || y1 != y2 {
			t.Errorf("%s not deterministic: first=(%v,%v) second=(%v,%v)", name, x1, y1, x2, y2)
		}
	}
}

// TestPathFnsVaryAcrossSeeds asserts that different seeds produce different
// trajectories for the same effect — the variation contract. Without this,
// every instance of "Gather" would look identical.
func TestPathFnsVaryAcrossSeeds(t *testing.T) {
	base := PathArgs{
		Elapsed:   200 * time.Millisecond,
		Duration:  800 * time.Millisecond,
		CellIdx:   3,
		CellCount: 10,
		OriginX:   0,
		OriginY:   0,
		TargetX:   20,
		TargetY:   5,
	}
	fns := map[string]PathFn{
		"Scatter": Scatter(4),
		"Pluck":   Pluck(),
		"Gather":  Gather(),
		"Vibrate": Vibrate(0.5),
		"Drift":   Drift(math.Pi/2, 5),
	}
	for name, fn := range fns {
		a := base
		a.Seed = 100
		b := base
		b.Seed = 200
		ax, ay := fn(a)
		bx, by := fn(b)
		if ax == bx && ay == by {
			t.Errorf("%s identical across seeds — variation contract violated", name)
		}
	}
}

// TestPathFnsAtZeroElapsed asserts every PathFn returns ~(0,0) at elapsed=0.
// Cells should start at their natural position; motion is what the function
// produces over time.
func TestPathFnsAtZeroElapsed(t *testing.T) {
	args := PathArgs{
		Elapsed:   0,
		Duration:  800 * time.Millisecond,
		CellIdx:   3,
		CellCount: 10,
		Seed:      42,
	}
	for name, fn := range map[string]PathFn{
		"Pluck":   Pluck(),
		"Gather":  Gather(),
		"Drift":   Drift(math.Pi/2, 5),
	} {
		x, y := fn(args)
		if math.Abs(x) > 0.5 || math.Abs(y) > 0.5 {
			t.Errorf("%s nonzero at elapsed=0: (%v, %v)", name, x, y)
		}
	}
}

// TestProgress asserts the Progress helper clamps elapsed/duration to [0,1].
func TestProgress(t *testing.T) {
	cases := []struct {
		elapsed, duration time.Duration
		want              float64
	}{
		{0, time.Second, 0},
		{500 * time.Millisecond, time.Second, 0.5},
		{time.Second, time.Second, 1.0},
		{2 * time.Second, time.Second, 1.0}, // clamps
		{-time.Second, time.Second, 0},      // clamps
	}
	for _, c := range cases {
		a := PathArgs{Elapsed: c.elapsed, Duration: c.duration}
		if got := a.Progress(); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Progress(%v/%v) = %v, want %v", c.elapsed, c.duration, got, c.want)
		}
	}
}
