package field

import (
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Note: the value-noise grid renderer is retired. Backdrop.Render now
// returns all-spaces so the field-as-backdrop pattern across screens
// produces a transparent overlay. Foreground cascades on the legacy field
// engine no longer render — screens that need cascades will be migrated
// to the typo package (tasks #36-#39).

// Backdrop is a per-screen wrapper around an Engine. It exposes a tight
// surface so any Screen can embed one, pump it on a tick, and composite
// its existing rendered content over the field.
//
// Usage in a Screen:
//
//	type Screen struct {
//	    backdrop *field.Backdrop
//	    // ...
//	}
//
//	func (s *Screen) Init() tea.Cmd {
//	    return tea.Batch(/* existing */, field.TickCmd())
//	}
//
//	func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
//	    switch m := msg.(type) {
//	    case field.TickMsg:
//	        s.backdrop.Tick(time.Time(m))
//	        return s, field.TickCmd()
//	    case tea.MouseMsg:
//	        s.backdrop.SetCursor(float64(m.X), float64(m.Y))
//	    case tea.KeyMsg:
//	        s.backdrop.Pulse(0.04)
//	        // ...screen's own handling
//	    }
//	    return s, nil
//	}
//
// And in View, composite content rows over field rows for the body area:
//
//	field := strings.Split(s.backdrop.Render(width, height), "\n")
//	body := field.Composite(contentRows, field, height)
type Backdrop struct {
	engine *Engine
}

// NewBackdrop returns a backdrop with a fresh Engine.
func NewBackdrop() *Backdrop {
	return &Backdrop{engine: NewEngine()}
}

// Engine returns the underlying engine for screens that need finer control.
func (b *Backdrop) Engine() *Engine { return b.engine }

// Tick advances the engine state one frame. Call on TickMsg.
func (b *Backdrop) Tick(t time.Time) { b.engine.Tick(t) }

// Pulse bumps the motion accumulator from non-mouse activity (typing,
// message arrivals, navigation events).
func (b *Backdrop) Pulse(amount float64) { b.engine.Pulse(amount) }

// SetTier moves the underlying engine into one of five intensity tiers.
// See Engine.SetTier for the tier semantics.
func (b *Backdrop) SetTier(t int) { b.engine.SetTier(t) }

// Tier returns the current intensity tier.
func (b *Backdrop) Tier() int { return b.engine.Tier() }

// SetCursor records cursor cell coords from MouseMsg.
func (b *Backdrop) SetCursor(x, y float64) { b.engine.SetCursor(x, y) }

// Render returns an empty backdrop. The grid renderer is retired — the
// engine is being phased out in favor of the typo pipeline, where motion
// lives in actual UI content (chat lines, game elements, spotlight title)
// rather than a separate animated layer behind everything. Screens that
// still call this get all-space rows so the existing composite path is a
// no-op overlay. Individual screens migrate to typo per task #36-#39.
func (b *Backdrop) Render(width, height int) string {
	_ = b.engine // engine still ticks for any code that reads its state directly
	if width <= 0 || height <= 0 {
		return ""
	}
	row := strings.Repeat(" ", width)
	out := make([]string, height)
	for i := range out {
		out[i] = row
	}
	return strings.Join(out, "\n")
}

// SetForegroundLines hands a list of foreground text overlays to the engine.
// Cells on those lines render with the engine's foreground style (bright
// white) and participate in the cursor cascade. Persistent variant — for
// event-driven text that auto-decays, use AddCascade.
func (b *Backdrop) SetForegroundLines(lines []Line) {
	b.engine.SetForegroundLines(lines)
}

// AddCascade fires a triggered cascade line that lives for c.Decay seconds.
// Use this for chat joins, spotlight rotations, mod alerts — discrete
// moments the engine should react to.
func (b *Backdrop) AddCascade(c CascadeLine) {
	b.engine.AddCascade(c)
}

// TickMsg fires at ~60fps while a backdrop is active. Screens dispatch this
// through their own Update; each screen's Init schedules the first one.
type TickMsg time.Time

// TickCmd returns the next backdrop tick. Schedule from Screen.Init and
// re-schedule from Screen.Update on TickMsg.
func TickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// ansiRE strips ANSI SGR sequences for the row-emptiness check.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// RowIsEmpty returns true if a row has no visible character content (only
// whitespace and escape sequences). Used by Composite to decide whether to
// show the field row underneath.
func RowIsEmpty(row string) bool {
	stripped := ansiRE.ReplaceAllString(row, "")
	return strings.TrimSpace(stripped) == ""
}

// Composite returns a height-row string. Each row is the content row if it
// has visible content, otherwise the field row. Per-row composition — not
// per-cell — but good enough for chat scrollbacks and feed lists where rows
// are either content-filled or fully empty.
func Composite(contentRows, fieldRows []string, height int) string {
	out := make([]string, height)
	for i := 0; i < height; i++ {
		var c, f string
		if i < len(contentRows) {
			c = contentRows[i]
		}
		if i < len(fieldRows) {
			f = fieldRows[i]
		}
		if RowIsEmpty(c) {
			out[i] = f
		} else {
			out[i] = c
		}
	}
	return strings.Join(out, "\n")
}
