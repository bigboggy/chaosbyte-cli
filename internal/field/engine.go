// Package field implements the value-noise-warped bitmap field engine,
// reverse-engineered from ertdfgcvb.xyz's js.js. The engine drives an ambient
// background of warped glyphs that responds to mouse motion (palette drift)
// and cursor proximity (per-cell cascade on foreground text overlays).
//
// Public API:
//
//	e := field.NewEngine()
//	e.Resize(width, height)
//	e.SetSourceWord("ERT")           // the 3-char bitmap source
//	e.SetForegroundLines([]field.Line{...})
//
// Per frame:
//
//	e.SetCursor(x, y)                // optional: feeds cursor halo + motion
//	e.Tick(time.Now())
//	out := e.Render()
package field

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	cellAspect = 0.5 // terminal cells are roughly 2 rows tall per column wide
)

// Default foregrounds (V.a and V.b in the engine). Muted on purpose.
var (
	defaultFgA = rgb{120, 120, 120}
	defaultFgB = rgb{90, 90, 90}
)

// Palette split into bright and dim pools so qA (highlight) always comes from
// bright and qB (ambient) always from dim. Guarantees a luminosity-contrasted
// pair on every motion burst.
var paletteBright = []rgb{
	{0xFF, 0xFF, 0xFF},
	{0xFF, 0x55, 0xFF},
	{0xFF, 0x55, 0xD5},
	{0x55, 0xD5, 0xFF},
}

var paletteDim = []rgb{
	{0, 0xAA, 0xAA},
	{0xAA, 0, 0},
	{0xAA, 0x55, 0},
	{0x55, 0x55, 0xFF},
	{0x55, 0x95, 0x55},
	{0xFF, 0x55, 0x55},
}

type rgb struct{ r, g, b uint8 }

func (c rgb) lipgloss() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b))
}

func lerpRGB(a, b rgb, t float64) rgb {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return rgb{
		r: uint8(float64(a.r)*(1-t) + float64(b.r)*t),
		g: uint8(float64(a.g)*(1-t) + float64(b.g)*t),
		b: uint8(float64(a.b)*(1-t) + float64(b.b)*t),
	}
}

func smoothstep01(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return x * x * (3 - 2*x)
}

// Two short ramps for the field's checkerboard rendering. The LAST character
// of each is dynamic: on motion saturation, the shapeN index reshuffles and
// the tails are updated to swap the densest glyph of the field.
var (
	rampA = []rune(" .·•-+=:;*ABC0123!*")
	rampB = []rune(" ·-•~+:*abcXYZ*")
)

// longRamp is the full glyph alphabet used for foreground cells cycling
// toward their target. Includes π, à, ò, ü, and / which are part of the
// engine's j ramp.
var longRamp = []rune(` .,·-•─~+:;=*π'"┐┌┘└┼├┤┴┬│╗╔╝╚╬╠╣╩╦║░▒▓█▄▀▌▐■!?&#$@/aàbcdefghijklmnoòpqrstuüvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789%()`)

// rampIdx maps each rune in longRamp to its index. Used to find the target
// landing position for cycling cells. 255 = "not in ramp, do not cycle".
var rampIdx map[rune]uint8

// shapePairs is the engine's A array: each pair drives the LAST glyph of
// the two short ramps after a motion-saturation reshuffle.
var shapePairs = [][2]rune{
	{' ', ' '},
	{'+', ' '},
	{' ', '.'},
	{'+', ' '},
	{' ', ','},
	{'·', ' '},
	{':', ' '},
	{'•', ' '},
}

// Value-noise tables. Permutation + value table, both 256 entries.
var (
	noiseTable [256]float64
	perm       [512]int
)

func init() {
	rampIdx = make(map[rune]uint8, len(longRamp))
	for i, r := range longRamp {
		if _, exists := rampIdx[r]; !exists {
			rampIdx[r] = uint8(i)
		}
	}

	r := rand.New(rand.NewSource(42))
	for i := 0; i < 256; i++ {
		noiseTable[i] = r.Float64()
		perm[i] = i
	}
	for i := 0; i < 256; i++ {
		j := r.Intn(256)
		perm[i], perm[j] = perm[j], perm[i]
	}
	for i := 0; i < 256; i++ {
		perm[i+256] = perm[i]
	}
}

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }
func mix(a, b, t float64) float64  { return a*(1-t) + b*t }

func valueNoise(px, py float64) float64 {
	xi := int(math.Floor(px))
	yi := int(math.Floor(py))
	tx := px - float64(xi)
	ty := py - float64(yi)
	rx0 := ((xi % 256) + 256) % 256
	rx1 := (rx0 + 1) % 256
	ry0 := ((yi % 256) + 256) % 256
	ry1 := (ry0 + 1) % 256
	c00 := noiseTable[perm[perm[rx0]+ry0]]
	c10 := noiseTable[perm[perm[rx1]+ry0]]
	c01 := noiseTable[perm[perm[rx0]+ry1]]
	c11 := noiseTable[perm[perm[rx1]+ry1]]
	sx := smoothstep(tx)
	sy := smoothstep(ty)
	nx0 := mix(c00, c10, sx)
	nx1 := mix(c01, c11, sx)
	return mix(nx0, nx1, sy)
}

func sampleBitmap(bmp []bool, w, h int, x, y float64) float64 {
	if x < 0 || y < 0 || x >= float64(w) || y >= float64(h) {
		return 0
	}
	xi := int(math.Floor(x))
	yi := int(math.Floor(y))
	x1 := xi + 1
	y1 := yi + 1
	if x1 >= w {
		x1 = w - 1
	}
	if y1 >= h {
		y1 = h - 1
	}
	tx := x - float64(xi)
	ty := y - float64(yi)
	idx := func(xx, yy int) float64 {
		if bmp[yy*w+xx] {
			return 1
		}
		return 0
	}
	a := mix(idx(xi, yi), idx(x1, yi), tx)
	b := mix(idx(xi, y1), idx(x1, y1), tx)
	return mix(a, b, ty)
}

func bitmapNearest(bmp []bool, w, h int, x, y float64) float64 {
	xi := int(math.Floor(x))
	yi := int(math.Floor(y))
	if xi < 0 || yi < 0 || xi >= w || yi >= h {
		return 0
	}
	if bmp[yi*w+xi] {
		return 1
	}
	return 0
}

// Line is a foreground text overlay placed at a specific row. Each character
// of Text becomes a cell with the cursor-driven cascade behavior.
type Line struct {
	Row      int
	Text     string
	HoverPad int // unused for now; reserved for per-line halo tuning
}

// flapState is the per-cell foreground cascade state.
type flapState struct {
	target    rune
	targetIdx uint8 // 255 if not in longRamp
	it        uint8 // 0 = idle (show target), 1 = cycling (show longRamp[lt])
	lt        uint8
}

// Engine is the field engine. Hold one per Screen instance.
type Engine struct {
	width, height int

	startedAt time.Time
	timeMs    float64
	frame     int64

	cursorX, cursorY float64
	prevX, prevY     float64
	halo             float64

	ot     float64
	qA, qB rgb
	rng    *rand.Rand

	shapeN int

	sourceWord string
	bmp        []bool
	bmpW, bmpH int

	fgLines []Line
	fgFlap  map[[2]int]*flapState

	// per-cell decayed intensity (motion-blur trails)
	fieldNT []float64
	gridLen int

	// Intensity tier 0..4, see SetTier.
	tier    int
	otFloor float64

	// cascades are time-bound foreground lines added via AddCascade. The
	// engine renders them on top of the field for cascade.Decay seconds
	// after BornAt, then drops them on the next Tick.
	cascades []CascadeLine
}

// CascadeLine is a foreground text overlay with a built-in lifespan. Unlike
// SetForegroundLines (which is for persistent overlays), cascades are
// event-triggered: a chat join, a spotlight rotation, a mod alert.
type CascadeLine struct {
	Row    int
	Text   string
	BornAt time.Time
	Decay  time.Duration
}

// NewEngine returns a configured engine with a default source word and
// freshly picked palette pair.
func NewEngine() *Engine {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bmp, w, h := composeWord("ERT")
	return &Engine{
		startedAt:  time.Now(),
		sourceWord: "ERT",
		bmp:        bmp,
		bmpW:       w,
		bmpH:       h,
		fgFlap:     make(map[[2]int]*flapState),
		qA:         paletteBright[rng.Intn(len(paletteBright))],
		qB:         paletteDim[rng.Intn(len(paletteDim))],
		rng:        rng,
		shapeN:     rng.Intn(len(shapePairs)),
	}
}

// Resize informs the engine of new terminal dimensions. Re-allocates the
// per-cell decayed-intensity buffer.
func (e *Engine) Resize(width, height int) {
	e.width = width
	e.height = height
	n := width * height
	if n != e.gridLen {
		e.gridLen = n
		e.fieldNT = make([]float64, n)
	}
}

// SetCursor records the cursor's current cell position. Pass once per
// MouseMsg.
func (e *Engine) SetCursor(x, y float64) {
	e.cursorX = x
	e.cursorY = y
}

// SetTier moves the engine into one of five intensity tiers. The mod calls
// this to express the room's energy:
//   - 0 Quiet ambient: idle baseline, palette barely drifts
//   - 1 Reactive: default — typing and cursor cause normal field response
//   - 2 Eventful: spotlight starts, nick joins, repo shared
//   - 3 Hype: blitz announcement, shoutout, parallel moments
//   - 4 Game / takeover: field recedes so the game has the floor
//
// The change takes effect on the next Tick; the floor is a soft minimum on
// the motion accumulator, not a fixed value.
func (e *Engine) SetTier(t int) {
	if t < 0 {
		t = 0
	}
	if t > 4 {
		t = 4
	}
	e.tier = t
	switch t {
	case 0:
		e.otFloor = 0
	case 1:
		e.otFloor = 0.15
	case 2:
		e.otFloor = 0.35
	case 3:
		e.otFloor = 0.65
	case 4:
		e.otFloor = 0
	}
}

// Tier returns the current intensity tier.
func (e *Engine) Tier() int { return e.tier }

// Pulse bumps the motion accumulator directly. Use this from non-mouse
// activity (keystrokes, message arrivals) to drive palette drift even when
// the cursor isn't moving. amount is added to ot; typical values 0.05-0.2.
func (e *Engine) Pulse(amount float64) {
	e.ot += amount
	if e.ot > 1 {
		e.ot = 1
	}
}

// SetSourceWord replaces the 3-char source bitmap. Pass uppercase letters
// or digits; anything else renders blank.
func (e *Engine) SetSourceWord(word string) {
	e.sourceWord = word
	e.bmp, e.bmpW, e.bmpH = composeWord(word)
}

// SourceWord returns the current 3-char source word.
func (e *Engine) SourceWord() string { return e.sourceWord }

// SetForegroundLines configures the text overlays the engine renders on top
// of the field. Each line is a row-positioned text that cascades when the
// cursor passes near it. This is the persistent variant; for event-driven
// text that auto-decays, use AddCascade instead.
func (e *Engine) SetForegroundLines(lines []Line) {
	e.fgLines = lines
	// Reset per-cell flap state so removed lines stop ghosting.
	e.fgFlap = make(map[[2]int]*flapState)
}

// updateForegroundLines replaces fgLines but preserves per-cell flap state
// for cells whose target rune at the same (row, col) hasn't changed. Used
// by AddCascade so adding/removing one cascade doesn't reset the others.
func (e *Engine) updateForegroundLines(lines []Line) {
	e.fgLines = lines
	targets := map[[2]int]rune{}
	for _, line := range lines {
		col := 0
		for _, r := range line.Text {
			targets[[2]int{col, line.Row}] = r
			col++
		}
	}
	for key, fl := range e.fgFlap {
		target, ok := targets[key]
		if !ok || target != fl.target {
			delete(e.fgFlap, key)
		}
	}
}

// AddCascade registers a time-bound foreground line. The engine pulses the
// motion accumulator on add so the cells actually flap into place, then
// drops the line after c.Decay. Replaces any active cascade on the same row.
//
// Use this for triggered moments: chat joins, spotlight rotation, mod
// alerts. The persistent SetForegroundLines is for fixed identity (e.g. an
// ambient screen's static labels).
func (e *Engine) AddCascade(c CascadeLine) {
	if c.BornAt.IsZero() {
		c.BornAt = time.Now()
	}
	if c.Decay == 0 {
		c.Decay = 4 * time.Second
	}
	placed := false
	for i := range e.cascades {
		if e.cascades[i].Row == c.Row {
			e.cascades[i] = c
			placed = true
			break
		}
	}
	if !placed {
		e.cascades = append(e.cascades, c)
	}
	e.rebuildCascadeLines()
	e.Pulse(0.95)
}

// rebuildCascadeLines collapses the active cascades into fgLines.
func (e *Engine) rebuildCascadeLines() {
	lines := make([]Line, 0, len(e.cascades))
	for _, c := range e.cascades {
		lines = append(lines, Line{Row: c.Row, Text: c.Text})
	}
	e.updateForegroundLines(lines)
}

// expireCascades drops cascades whose lifespan is over. Called once per Tick.
func (e *Engine) expireCascades(now time.Time) {
	fresh := e.cascades[:0]
	changed := false
	for _, c := range e.cascades {
		if now.Sub(c.BornAt) < c.Decay {
			fresh = append(fresh, c)
		} else {
			changed = true
		}
	}
	if changed {
		e.cascades = fresh
		e.rebuildCascadeLines()
	}
}

// Tick advances the engine's state one frame. Call before Render.
func (e *Engine) Tick(now time.Time) {
	e.expireCascades(now)
	e.timeMs = float64(now.Sub(e.startedAt).Milliseconds())
	e.frame++

	dx := e.cursorX - e.prevX
	dy := (e.cursorY - e.prevY) / cellAspect
	speed := math.Sqrt(dx*dx + dy*dy)

	e.halo *= 0.75
	e.halo += speed * 0.4
	if e.halo > 35 {
		e.halo = 35
	}

	var otGrowth float64
	if speed > 0.1 {
		otGrowth = 0.008
	}
	e.ot += otGrowth
	e.ot -= 0.003
	if e.ot < 0 {
		e.ot = 0
	}
	if e.ot > 1 {
		e.ot = 1
	}
	// Intensity tier floor: the mod can push the engine into higher tiers
	// without supplying constant motion; the floor keeps ot from sagging
	// below tier-appropriate baseline. Tier 4 overrides everything to zero
	// so the field recedes during game / takeover moments.
	if e.tier == 4 {
		e.ot = 0
	} else if e.ot < e.otFloor {
		e.ot = e.otFloor
	}

	// Tier 0 is meant to be quiet — no per-frame palette reshuffle so the
	// field stops flickering when nothing's happening in the room. The
	// palette pair is whatever the engine last picked at tier 1+, then
	// freezes here until something raises the tier or fires a cascade.
	if e.tier > 0 && e.ot < 0.3 {
		e.qA = paletteBright[e.rng.Intn(len(paletteBright))]
		e.qB = paletteDim[e.rng.Intn(len(paletteDim))]
	}

	if e.ot >= 0.99 {
		e.shapeN = e.rng.Intn(len(shapePairs))
	}
	updateRampTails(e.shapeN)

	e.prevX = e.cursorX
	e.prevY = e.cursorY

	// Foreground cascade per-cell: re-stamp on halo proximity, advance after.
	rampLen := uint8(len(longRamp))
	if len(e.fgLines) > 0 {
		rtSq := e.halo
		cx := e.cursorX
		cy := e.cursorY
		cols := e.width
		for _, line := range e.fgLines {
			row := line.Row
			col := cols/2 - len(line.Text)/2
			for i, r := range line.Text {
				key := [2]int{col + i, row}
				st, ok := e.fgFlap[key]
				if !ok {
					st = &flapState{target: r}
					if idx, ok := rampIdx[r]; ok {
						st.targetIdx = idx
					} else {
						st.targetIdx = 255
					}
					e.fgFlap[key] = st
				}
				if st.targetIdx == 255 {
					continue
				}
				ddx := float64(col+i) - cx
				ddy := (float64(row) - cy) / cellAspect
				distSq := ddx*ddx + ddy*ddy

				if rtSq > 0.5 && distSq < rtSq {
					depth := rtSq - distSq
					if depth < 0 {
						depth = 0
					}
					st.lt = uint8(int(depth) % len(longRamp))
					st.it = 1
				} else if st.it == 1 {
					// Advance every other frame for smoother visible transitions.
					if e.frame%2 == 0 {
						if st.lt == st.targetIdx {
							st.it = 0
						} else {
							st.lt = (st.lt + 1) % rampLen
						}
					}
				}
			}
		}
	}
}

func updateRampTails(n int) {
	pair := shapePairs[n%len(shapePairs)]
	if idx, ok := rampIdx[pair[0]]; ok {
		next := (int(idx) + 1) % len(longRamp)
		rampA[len(rampA)-1] = longRamp[next]
	}
	if idx, ok := rampIdx[pair[1]]; ok {
		next := (int(idx) + 1) % len(longRamp)
		rampB[len(rampB)-1] = longRamp[next]
	}
}

// fgCell is the resolved cell info for a foreground overlay position.
type fgCell struct {
	ch rune
}

// Render produces the current frame as a styled multi-line string. Cell
// styling is batched per same-style run within each row.
func (e *Engine) Render() string {
	if e.width == 0 || e.height == 0 {
		return ""
	}
	cols := e.width
	rows := e.height

	bmpAspect := (float64(e.bmpW) / float64(e.bmpH)) / cellAspect
	scrAspect := float64(cols) / float64(rows)
	var scaleX, scaleY float64
	if bmpAspect < scrAspect {
		scaleY = 1.0 / float64(rows)
		scaleX = 1.0 / float64(rows) / bmpAspect
	} else {
		scaleX = 1.0 / float64(cols)
		scaleY = 1.0 / float64(cols) * bmpAspect
	}

	t := e.timeMs * 0.0004
	uOsc := 0.85 - 0.35*math.Cos(e.timeMs*0.0004)

	stage1 := smoothstep01(e.ot / 0.5)
	stage2 := smoothstep01((e.ot - 0.5) / 0.5)
	fieldA := lerpRGB(defaultFgA, e.qA, stage1)
	fieldB := lerpRGB(defaultFgB, e.qB, stage1)
	white := rgb{255, 255, 255}
	fieldA = lerpRGB(fieldA, white, stage2*0.6)
	fieldB = lerpRGB(fieldB, white, stage2*0.4)

	baseStyle := lipgloss.NewStyle().Foreground(fieldB.lipgloss())
	hiStyle := lipgloss.NewStyle().Foreground(fieldA.lipgloss())
	fgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
	styles := [3]lipgloss.Style{baseStyle, hiStyle, fgStyle}
	const (
		styleBase = 0
		styleHi   = 1
		styleFg   = 2
	)

	fgGrid := make(map[[2]int]fgCell)
	for _, line := range e.fgLines {
		row := line.Row
		col := cols/2 - len(line.Text)/2
		for i, r := range line.Text {
			key := [2]int{col + i, row}
			cell := fgCell{ch: r}
			if st, ok := e.fgFlap[key]; ok && st.it == 1 {
				cell.ch = longRamp[st.lt]
			}
			fgGrid[key] = cell
		}
	}

	var b strings.Builder
	b.Grow(cols * rows * 8)

	var runBuf strings.Builder
	runStyle := -1
	flushRun := func() {
		if runStyle >= 0 && runBuf.Len() > 0 {
			b.WriteString(styles[runStyle].Render(runBuf.String()))
			runBuf.Reset()
		}
	}

	for y := 0; y < rows; y++ {
		runStyle = -1
		runBuf.Reset()
		for x := 0; x < cols; x++ {
			var ch rune
			var stID int

			if cell, ok := fgGrid[[2]int{x, y}]; ok {
				stID = styleFg
				ch = cell.ch
			} else {
				cx := (float64(x)-float64(cols)/2)*scaleX + 0.5
				cy := (float64(y)-float64(rows)/2)*scaleY + 0.5

				wx := cx + 0.5*(valueNoise(cx*uOsc+t, cy*uOsc)-0.5)
				wy := cy + 1.8*(valueNoise(cx*uOsc, cy*uOsc+t)-0.5)

				ft := math.Max(
					sampleBitmap(e.bmp, e.bmpW, e.bmpH, wx*float64(e.bmpW), wy*float64(e.bmpH)),
					bitmapNearest(e.bmp, e.bmpW, e.bmpH, wx*float64(e.bmpW), wy*float64(e.bmpH)),
				)
				cellIdx := y*e.width + x
				var nt float64
				if cellIdx >= 0 && cellIdx < len(e.fieldNT) {
					nt = e.fieldNT[cellIdx]
				}
				v := ft
				if nt > v {
					v = nt
				}
				if cellIdx >= 0 && cellIdx < len(e.fieldNT) {
					e.fieldNT[cellIdx] = 0.95 * v
				}

				ramp := rampA
				if (x+y)%2 == 1 {
					ramp = rampB
				}
				idx := int(v * float64(len(ramp)-1))
				if idx < 0 {
					idx = 0
				}
				if idx >= len(ramp) {
					idx = len(ramp) - 1
				}
				ch = ramp[idx]
				if v >= 0.99 {
					stID = styleHi
				} else {
					stID = styleBase
				}
			}

			if stID != runStyle {
				flushRun()
				runStyle = stID
			}
			runBuf.WriteRune(ch)
		}
		flushRun()
		if y < rows-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
