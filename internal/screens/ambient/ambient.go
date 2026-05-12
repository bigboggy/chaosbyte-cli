// Package ambient is the field-driven backdrop screen. It owns a field.Engine
// and pumps it on a 60fps tick. Mouse motion and key input flow through to
// the engine; the rendered field fills the screen body.
//
// Mouse-driven cursor cascade requires the program to be started with
// tea.WithMouseAllMotion. The default chaosbyte main.go uses
// tea.WithMouseCellMotion, which only reports motion while a button is held;
// in that mode the cursor cascade only triggers on click-drag. The engine
// still runs (palette drift, warp breathing, shape variation) without any
// cursor input.
package ambient

import (
	"time"
	"unicode"

	"github.com/bchayka/gitstatus/internal/field"
	"github.com/bchayka/gitstatus/internal/screens"
	tea "github.com/charmbracelet/bubbletea"
)

// TickMsg fires at ~60fps while ambient is on screen.
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Screen is the ambient backdrop. It holds the field engine and a small bit
// of input state (the partial word the user is typing to change the source
// bitmap).
type Screen struct {
	engine *field.Engine
	width  int
	height int

	wordInput string // 0-3 chars; commits when length hits 3
}

// New returns an ambient screen seeded with the default source word and a
// short instructional foreground.
func New() *Screen {
	e := field.NewEngine()
	e.SetForegroundLines(defaultLines("ERT", ""))
	return &Screen{engine: e}
}

func defaultLines(currentWord, typing string) []field.Line {
	status := "source: " + currentWord
	if typing != "" {
		status = "typing: " + typing + "_"
	}
	return []field.Line{
		{Row: 2, Text: "chaosbyte / ambient"},
		{Row: 4, Text: status},
		{Row: 6, Text: "type 3 letters to change the source word"},
		{Row: 7, Text: "move your cursor over text to flap it"},
		{Row: 8, Text: "esc returns to lobby"},
	}
}

func (s *Screen) Init() tea.Cmd { return tickCmd() }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height
		s.engine.Resize(m.Width, m.Height)
		return s, nil

	case tea.MouseMsg:
		s.engine.SetCursor(float64(m.X), float64(m.Y))
		return s, nil

	case tea.KeyMsg:
		switch m.Type {
		case tea.KeyEsc:
			return s, screens.Navigate(screens.LobbyID)
		case tea.KeyBackspace:
			if len(s.wordInput) > 0 {
				s.wordInput = s.wordInput[:len(s.wordInput)-1]
				s.engine.SetForegroundLines(defaultLines(s.engine.SourceWord(), s.wordInput))
			}
			return s, nil
		case tea.KeyEnter:
			if len(s.wordInput) > 0 {
				s.commitWord(padOrTrim(s.wordInput, 3))
			}
			return s, nil
		case tea.KeyRunes:
			for _, r := range m.Runes {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					if len(s.wordInput) < 3 {
						s.wordInput += string(unicode.ToUpper(r))
					}
				}
			}
			if len(s.wordInput) == 3 {
				s.commitWord(s.wordInput)
			} else {
				s.engine.SetForegroundLines(defaultLines(s.engine.SourceWord(), s.wordInput))
			}
			return s, nil
		}

	case TickMsg:
		s.engine.Tick(time.Time(m))
		return s, tickCmd()
	}
	return s, nil
}

func (s *Screen) commitWord(w string) {
	s.engine.SetSourceWord(w)
	s.wordInput = ""
	s.engine.SetForegroundLines(defaultLines(w, ""))
}

func padOrTrim(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	for len(s) < n {
		s += " "
	}
	return s
}

func (s *Screen) View(width, height int) string {
	// Engine renders at its own resized dimensions. The chrome (header/footer)
	// is composited by the app; this returns only the body.
	return s.engine.Render()
}

func (s *Screen) Name() string          { return screens.AmbientID }
func (s *Screen) Title() string         { return "ambient" }
func (s *Screen) HeaderContext() string { return s.engine.SourceWord() }

func (s *Screen) Footer() []screens.KeyHint {
	return []screens.KeyHint{
		{Key: "a-z 0-9", Desc: "change word"},
		{Key: "mouse", Desc: "flap text"},
		{Key: "esc", Desc: "lobby"},
	}
}

// InputFocused returns true while the user is mid-word. Without this, the
// global key handlers would intercept letters/digits as shortcuts.
func (s *Screen) InputFocused() bool { return s.wordInput != "" }
