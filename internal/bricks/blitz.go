package bricks

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BlitzStartedMsg fires once when the round begins so the host can suppress
// its own input handling and announce the start in chat.
type BlitzStartedMsg struct{}

// BlitzEndedMsg fires once after the round resolves. The host owns what
// happens next: back to lobby, post a score, queue the next blitz.
type BlitzEndedMsg struct {
	TopScore int
	Lines    int
}

type blitzTickMsg time.Time

const blitzFrameRate = 30 * time.Millisecond

// BricksBlitz is the self-contained game widget.
type BricksBlitz struct {
	width  int
	height int

	state *state
	ended bool
}

// NewBricksBlitz returns a fresh widget sized for the surrounding pane.
// seedLines feeds the bar pool — pass recent chat or a curated set. Entries
// outside the 5..12 char range are normalised or dropped before play.
func NewBricksBlitz(width, height int, seedLines []string) *BricksBlitz {
	return &BricksBlitz{
		width:  width,
		height: height,
		state:  newState(seedLines),
	}
}

// Init starts the round. The first tickMsg arrives one frame later, which
// is when the state machine flips to running.
func (b *BricksBlitz) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return BlitzStartedMsg{} },
		blitzTick(),
	)
}

func blitzTick() tea.Cmd {
	return tea.Tick(blitzFrameRate, func(t time.Time) tea.Msg {
		return blitzTickMsg(t)
	})
}

// Update advances the simulation. The host forwards every tea.Msg here
// until Done() returns true.
func (b *BricksBlitz) Update(msg tea.Msg) (*BricksBlitz, tea.Cmd) {
	if b.ended {
		return b, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		return b, nil

	case blitzTickMsg:
		now := time.Time(msg)
		if b.state.phase == phaseReady {
			b.state.start(now)
		}
		over := b.state.advance(now)
		if over {
			b.ended = true
			score, lines := b.Score()
			return b, func() tea.Msg {
				return BlitzEndedMsg{TopScore: score, Lines: lines}
			}
		}
		return b, blitzTick()

	case tea.KeyMsg:
		return b.handleKey(msg)
	}
	return b, nil
}

func (b *BricksBlitz) handleKey(msg tea.KeyMsg) (*BricksBlitz, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		b.state.movePaddle(-3)
	case "right", "l":
		b.state.movePaddle(3)
	case "shift+left", "H":
		b.state.movePaddle(-6)
	case "shift+right", "L":
		b.state.movePaddle(6)
	}
	return b, nil
}

// Resize adjusts the rendering box. The simulation grid stays fixed.
func (b *BricksBlitz) Resize(width, height int) {
	b.width = width
	b.height = height
}

// View renders the playfield, HUD and (when applicable) the end card.
func (b *BricksBlitz) View() string { return b.renderView() }

// Done reports whether the round has finished. After Done returns true the
// host should stop forwarding messages and read the final Score.
func (b *BricksBlitz) Done() bool { return b.ended }

// Score returns the in-progress or final score, plus the number of bars hit.
func (b *BricksBlitz) Score() (points int, linesHit int) {
	return b.state.score, b.state.hits
}
