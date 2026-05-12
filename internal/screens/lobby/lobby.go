// Package lobby is the chat-style entry point that doubles as the app's home
// screen. It owns the channel list, manages an always-focused input, and
// routes slash commands to other screens via screens.Navigate.
//
// Files in this package:
//   - lobby.go     — Screen type, Init/Update/View, message posting
//   - commands.go  — slash command registry + per-command handlers
//   - completion.go — Tab autocomplete
//   - seed.go      — fake channels + the @boggy username
package lobby

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/field"
	"github.com/bchayka/gitstatus/internal/mod"
	"github.com/bchayka/gitstatus/internal/room"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/typo"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen is the lobby's own state. The chat input is always focused, so this
// screen reports InputFocused()==true and the app's global key handlers stay
// out of the way.
type Screen struct {
	nick string

	channels   []Channel
	chatActive int
	chatScroll int

	input      textinput.Model
	history    []string
	historyIdx int
	paletteIdx int // selection inside the command palette popup

	joinPosted bool

	// backdrop drives the field engine behind the chat scrollback. Each
	// keystroke pulses its motion accumulator so typing produces palette
	// drift the same way mouse motion does on the ertdfgcvb site.
	backdrop *field.Backdrop

	// tier state: hypeUntil holds the deadline for a tier-3 burst triggered
	// by chat arrivals; lastChatEvent is used to drop to tier 0 on long
	// silences.
	hypeUntil     time.Time
	lastChatEvent time.Time

	mod *mod.Mod

	// broker hands the shared #lobby scrollback across SSH sessions. When
	// nil the lobby runs fully local — useful for `go run .` and tests.
	broker   *room.Broker
	roomSub  <-chan room.Event
	lobbyIdx int

	// choreographer drives event-triggered cell animations (waves, awards,
	// gathers). Lobby renders any active CellTransforms over the chat
	// scrollback area each frame.
	choreographer    *typo.Choreographer
	activeTransforms []typo.CellTransform // updated each Tick; consumed in View
	lastPlacements   []msgPlacement       // chat message body Layouts + positions from last View
}

// msgPlacement records where one chat message's body lives in the rendered
// scrollback. /wave + future event handlers use this to schedule transforms
// on real chat cells.
type msgPlacement struct {
	Layout       *typo.Layout
	Body         string
	PrefixWidth  int
	FlatRowStart int
}

// New constructs a fresh lobby with seeded channels and a focused input.
// nick is the user's chat handle; broker is the shared room state and may
// be nil for fully-local mode. When broker is attached every channel's
// scrollback comes from broker.Snapshot and a single subscription delivers
// events for all channels.
func New(nick string, broker *room.Broker) *Screen {
	if nick == "" {
		nick = "@boggy"
	}
	s := &Screen{
		nick:          nick,
		channels:      seedChannels(),
		chatActive:    0,
		input:         newInput(),
		backdrop:      field.NewBackdrop(),
		mod:           mod.New(),
		broker:        broker,
		lobbyIdx:      -1,
		choreographer: typo.NewChoreographer(),
	}
	for i, ch := range s.channels {
		if ch.Name == "#lobby" {
			s.lobbyIdx = i
			break
		}
	}
	if broker != nil {
		for i, ch := range s.channels {
			if msgs := broker.Snapshot(ch.Name); len(msgs) > 0 {
				s.channels[i].Messages = msgs
			}
		}
		s.roomSub = broker.Subscribe()
	}
	return s
}

func newInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Placeholder = "message #lobby or type /help"
	ti.Focus()
	return ti
}

func (s *Screen) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, field.TickCmd()}
	if c := s.waitForRoom(); c != nil {
		cmds = append(cmds, c)
	}
	return tea.Batch(cmds...)
}

// roomEventMsg wraps a broker.Event so the lobby can handle it in Update.
type roomEventMsg room.Event

// waitForRoom returns the Bubbletea command that blocks on the broker
// subscription. Each broker event is delivered as one roomEventMsg; Update
// re-issues the command to keep listening.
func (s *Screen) waitForRoom() tea.Cmd {
	if s.roomSub == nil {
		return nil
	}
	sub := s.roomSub
	return func() tea.Msg {
		evt, ok := <-sub
		if !ok {
			return nil
		}
		return roomEventMsg(evt)
	}
}

func (s *Screen) Name() string  { return screens.LobbyID }
func (s *Screen) Title() string { return "lobby" }

func (s *Screen) HeaderContext() string {
	ch := s.activeChannel()
	if ch == nil {
		return ""
	}
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	return lipgloss.NewStyle().Foreground(theme.OK).Render(ch.Name) + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("%d online", ch.Online))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.paletteVisible() {
		return []screens.KeyHint{
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "tab", Desc: "fill"},
			{Key: "enter", Desc: "run"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	return []screens.KeyHint{
		{Key: "enter", Desc: "send"},
		{Key: "/", Desc: "commands"},
		{Key: "↑/↓", Desc: "history"},
		{Key: "pgup/pgdn", Desc: "scroll"},
		{Key: "ctrl+c", Desc: "quit"},
	}
}

func (s *Screen) InputFocused() bool { return true }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		s.backdrop.Pulse(0.04)
		return s.handleKey(m)
	case tea.MouseMsg:
		s.backdrop.SetCursor(float64(m.X), float64(m.Y))
		return s, nil
	case roomEventMsg:
		s.handleRoomEvent(room.Event(m))
		s.hypeUntil = time.Now().Add(5 * time.Second)
		s.lastChatEvent = time.Now()
		return s, s.waitForRoom()
	case field.TickMsg:
		t := time.Time(m)
		s.backdrop.Tick(t)
		s.updateTier(t)
		s.activeTransforms = s.choreographer.Tick(t)
		if s.broker == nil {
			if line := s.mod.Tick(t); line != "" {
				s.postMod(line)
			}
		}
		return s, field.TickCmd()
	}
	return s, nil
}

func (s *Screen) handleKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "enter":
		// When the palette is open, Enter accepts the highlighted command —
		// it's inserted in the input then submitted in one keystroke, matching
		// the muscle memory of Claude Code / VS Code command palettes.
		if cmd := s.acceptPalette(); cmd != "" {
			s.input.SetValue(cmd)
		}
		s.resetPalette()
		return s.submit()
	case "tab":
		// Tab fills the input without submitting, so users can pick a command
		// like /join and then type the channel name.
		if s.paletteVisible() {
			s.fillPalette()
		}
		return s, nil
	case "shift+tab":
		if s.paletteVisible() {
			s.movePalette(-1)
		}
		return s, nil
	case "up":
		if s.paletteVisible() {
			s.movePalette(-1)
		} else {
			s.recallHistory(-1)
		}
		return s, nil
	case "down":
		if s.paletteVisible() {
			s.movePalette(+1)
		} else {
			s.recallHistory(+1)
		}
		return s, nil
	case "pgup":
		s.chatScroll += 5
		return s, nil
	case "pgdown":
		s.chatScroll -= 5
		if s.chatScroll < 0 {
			s.chatScroll = 0
		}
		return s, nil
	case "esc":
		s.input.SetValue("")
		s.resetPalette()
		return s, nil
	}
	// Any other key edits the input → the filtered match list will change, so
	// reset the highlight back to the top of the new list.
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.resetPalette()
	return s, cmd
}

func (s *Screen) submit() (screens.Screen, tea.Cmd) {
	text := strings.TrimSpace(s.input.Value())
	s.input.SetValue("")
	if text == "" {
		return s, nil
	}
	s.history = append(s.history, text)
	s.historyIdx = len(s.history)

	if strings.HasPrefix(text, "/") {
		ss, cmd := s.handleSlash(text)
		return ss, cmd
	}
	s.postUser(text)
	return s, nil
}

func (s *Screen) recallHistory(delta int) {
	if len(s.history) == 0 {
		return
	}
	switch {
	case delta < 0:
		if s.historyIdx > 0 {
			s.historyIdx--
		}
	case delta > 0:
		if s.historyIdx < len(s.history) {
			s.historyIdx++
		}
	}
	if s.historyIdx >= len(s.history) {
		s.input.SetValue("")
		return
	}
	s.input.SetValue(s.history[s.historyIdx])
	s.input.CursorEnd()
}

// ---------------------------------------------------------------------------
// Posting helpers — used by both regular sends and slash command handlers
// ---------------------------------------------------------------------------

func (s *Screen) activeChannel() *Channel {
	if s.chatActive < 0 || s.chatActive >= len(s.channels) {
		return nil
	}
	return &s.channels[s.chatActive]
}

func (s *Screen) postUser(body string) {
	ch := s.activeChannel()
	if ch == nil {
		return
	}
	now := time.Now()
	msg := ui.ChatMessage{Author: s.nick, Body: body, At: now}
	if s.broker != nil {
		s.broker.Publish(ch.Name, msg)
		return
	}
	ch.Messages = append(ch.Messages, msg)
	s.chatScroll = 0
	s.mod.NoteChat(now)
}

// updateTier maps room state onto the field's five intensity tiers. With a
// broker attached we defer to broker.Tier() so every screen sees the same
// energy level the mod sees; locally we fall back to a smaller idle/burst
// heuristic.
func (s *Screen) updateTier(t time.Time) {
	if s.broker != nil {
		s.backdrop.SetTier(s.broker.Tier())
		return
	}
	switch {
	case t.Before(s.hypeUntil):
		s.backdrop.SetTier(3)
	case !s.lastChatEvent.IsZero() && t.Sub(s.lastChatEvent) > 30*time.Second:
		s.backdrop.SetTier(0)
	default:
		s.backdrop.SetTier(1)
	}
}

// handleRoomEvent applies a broker event to the local channel mirror so the
// lobby renders the same scrollback every other session sees. Some kinds
// (joins, mod posts) also trigger a foreground cascade so the engine
// announces them visibly on top of the field.
func (s *Screen) handleRoomEvent(evt room.Event) {
	for i := range s.channels {
		if s.channels[i].Name != evt.Channel {
			continue
		}
		s.channels[i].Messages = append(s.channels[i].Messages, evt.Message)
		if i == s.chatActive {
			s.chatScroll = 0
		}
		break
	}
	if evt.Channel != s.activeChannelName() {
		return
	}
	switch {
	case evt.Message.Kind == ui.ChatJoin:
		s.backdrop.AddCascade(field.CascadeLine{
			Row:   0,
			Text:  evt.Message.Author + " joined",
			Decay: 4 * time.Second,
		})
	case evt.Message.Kind == ui.ChatAction && evt.Message.Author == mod.Nick:
		s.backdrop.AddCascade(field.CascadeLine{
			Row:   1,
			Text:  mod.Nick + " · " + evt.Message.Body,
			Decay: 5 * time.Second,
		})
	}
}

// activeChannelName returns the name of the active channel, or "" if none.
func (s *Screen) activeChannelName() string {
	if ch := s.activeChannel(); ch != nil {
		return ch.Name
	}
	return ""
}

// postMod posts a moderator line to the active channel. Visually identical
// to ChatAction (italic accent2) but with the @mod author convention.
func (s *Screen) postMod(body string) {
	ch := s.activeChannel()
	if ch == nil {
		return
	}
	now := time.Now()
	ch.Messages = append(ch.Messages, ui.ChatMessage{
		Author: mod.Nick, Body: body, At: now, Kind: ui.ChatAction,
	})
	s.chatScroll = 0
	s.mod.NoteChat(now)
}

func (s *Screen) postSystem(body string) {
	ch := s.activeChannel()
	if ch == nil {
		return
	}
	for _, line := range strings.Split(body, "\n") {
		ch.Messages = append(ch.Messages, ui.ChatMessage{
			Author: "*", Body: line, At: time.Now(), Kind: ui.ChatSystem,
		})
	}
	s.chatScroll = 0
}

// EnsureJoined posts the "entered the chat" join message once, then no-ops on
// subsequent calls. Called by the router when transitioning from intro. When
// a broker is attached the join broadcasts so other sessions see the new
// nick and the broker's mod auto-welcomes the room.
func (s *Screen) EnsureJoined() {
	if s.joinPosted {
		return
	}
	now := time.Now()
	joinMsg := ui.ChatMessage{
		Author: s.nick, Body: "entered the chat", At: now, Kind: ui.ChatJoin,
	}
	if s.broker != nil {
		s.broker.Publish("#lobby", joinMsg)
	} else if ch := s.activeChannel(); ch != nil {
		ch.Messages = append(ch.Messages, joinMsg)
		s.postMod(s.mod.Welcome(s.nick))
	}
	s.joinPosted = true
	s.chatScroll = 0
}

// OnEnter is the router's field-driven entry hook. We pulse the backdrop hard
// and register the user's nick as a foreground line so it flap-spins across
// the field; both decay naturally over a few seconds.
func (s *Screen) OnEnter() {
	s.backdrop.AddCascade(field.CascadeLine{
		Row:   0,
		Text:  s.nick + " · welcome to chaosbyte",
		Decay: 5 * time.Second,
	})
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	if s.chatActive < 0 || s.chatActive >= len(s.channels) {
		s.chatActive = 0
	}
	ch := s.channels[s.chatActive]

	bar := topBar(ch, contentW)
	barH := lipgloss.Height(bar)

	prompt := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
		Render("[" + strings.TrimPrefix(s.nick, "@") + "]> ")
	s.input.Width = contentW - lipgloss.Width(prompt) - 1
	inputLine := prompt + s.input.View()

	palette := s.renderPalette(contentW)
	paletteH := s.paletteHeight()

	// Layout (bottom-anchored input):
	//   bar (1) · divider (1) · scrollback (chatH) · divider (1) · palette (paletteH) · input (1)
	// Total fixed chrome is 4 rows; scrollback flexes around it.
	chatH := height - barH - paletteH - 4
	if chatH < 4 {
		chatH = 4
	}

	now := time.Now()

	// Build chat rows. Track each message's body Layout and where in the
	// flat row stream its body lands, so the choreographer can target real
	// chat cells (not phantom demo layouts) when an effect fires.
	var lines []string
	placements := make([]msgPlacement, 0, len(ch.Messages))
	for _, msg := range ch.Messages {
		rendered, body, prefixW := renderChatLineAnimDetailed(msg, contentW, now)
		flatStart := len(lines)
		lines = append(lines, rendered...)
		layoutID := msgKey(msg)
		layout := typo.Prepare(layoutID, body, contentW-prefixW-1)
		placements = append(placements, msgPlacement{
			Layout:       layout,
			Body:         body,
			PrefixWidth:  prefixW,
			FlatRowStart: flatStart,
		})
	}
	s.lastPlacements = placements
	chatRows := scrollbackRows(lines, chatH, s.chatScroll)

	// Field engine grid is no longer rendered behind chat — the substrate
	// is the chat itself, animated via the choreographer. Empty rows stay
	// empty; transforms render directly on top of the chat string.
	visible := strings.Join(chatRows, "\n")

	// Render active CellTransforms on top of the chat — borrowed cells
	// move across the chat area; the original positions are blanked so
	// you see the actual chat text travelling, not a phantom duplicate.
	if len(s.activeTransforms) > 0 {
		visible = s.composeTransformOverlay(visible, contentW, chatH, now, placements)
	}

	parts := []string{
		bar,
		ui.Divider(contentW),
		visible,
		ui.Divider(contentW),
	}
	if palette != "" {
		parts = append(parts, palette)
	}
	parts = append(parts, inputLine)
	stacked := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, stacked)
}

func topBar(ch Channel, width int) string {
	chName := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(ch.Name)
	online := lipgloss.NewStyle().Foreground(theme.OK).Render(fmt.Sprintf("%d online", ch.Online))
	topic := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("topic: " + ch.Topic)
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render("  ·  ")
	left := chName + sep + online + sep + topic
	if lipgloss.Width(left) > width {
		left = ui.Truncate(left, width)
	}
	return left
}

// composeTransformOverlay renders active CellTransforms over the chat
// scrollback. Origins are derived from the placements collected during
// chat-row construction so the borrowed cells move FROM their real
// chat-screen position OUTWARD, and back. The original cell positions
// are blanked in the base string so the user sees one set of moving
// chars, not a ghost duplicate.
func (s *Screen) composeTransformOverlay(base string, width, height int, now time.Time, placements []msgPlacement) string {
	// Resolve each Layout ID to its on-screen origin in chat-area coords.
	totalRows := 0
	for _, p := range placements {
		// Placement's body row count is the rendered rows minus 0 (prefix
		// is on row 0; we'll align the body to the same first row).
		totalRows += p.Layout.Height
	}
	// Top padding when chat is shorter than the area (bottom-aligned).
	topPad := 0
	if flat := totalFlatRows(placements); flat < height {
		topPad = height - flat
	}

	origins := map[string]typo.LayoutOrigin{}
	for _, p := range placements {
		y := topPad + p.FlatRowStart
		x := p.PrefixWidth + 1
		origins[p.Layout.ID] = typo.LayoutOrigin{Layout: p.Layout, X: x, Y: y}
	}

	// Build the overlay grid.
	comp := typo.NewCompositor(width, height)
	comp.DrawTransforms(s.activeTransforms, origins, now)
	overlay := comp.Render()

	// Blank the natural cell positions for any cell currently in transform
	// so the chat text appears to physically leave its spot.
	borrowedByLayout := typo.IndexTransforms(s.activeTransforms)
	base = blankBorrowedCells(base, origins, borrowedByLayout, height)

	return overlayRows(base, overlay, height)
}

// totalFlatRows sums the rendered heights of all placements (each Layout
// reports its own wrapped row count via Height).
func totalFlatRows(placements []msgPlacement) int {
	total := 0
	for _, p := range placements {
		total += p.Layout.Height
	}
	return total
}

// blankBorrowedCells walks each transformed cell, computes its natural
// (col, row) in the base chat string, and replaces that visible cell with
// a space. The transformed copy renders elsewhere via the overlay.
func blankBorrowedCells(base string, origins map[string]typo.LayoutOrigin, borrowed map[string]map[int]bool, height int) string {
	rows := splitToHeight(base, height)
	for layoutID, idxSet := range borrowed {
		origin, ok := origins[layoutID]
		if !ok {
			continue
		}
		for idx := range idxSet {
			if idx < 0 || idx >= len(origin.Layout.Cells) {
				continue
			}
			cell := origin.Layout.Cells[idx]
			col := origin.X + cell.Col
			row := origin.Y + cell.Row
			if row < 0 || row >= len(rows) {
				continue
			}
			rows[row] = blankCellAt(rows[row], col)
		}
	}
	return strings.Join(rows, "\n")
}

// blankCellAt replaces the visible char at the given column with a space,
// preserving any ANSI styling on either side. Implemented as a simple
// strip-and-rebuild for correctness over speed.
func blankCellAt(row string, col int) string {
	plain := stripAnsi(row)
	runes := []rune(plain)
	if col < 0 || col >= len(runes) {
		return row
	}
	runes[col] = ' '
	return string(runes)
}

// overlayRows merges two equal-height multi-line strings, taking the
// overlay's non-blank cells in preference to base. Both strings are
// expected to have `height` rows; if not, the shorter one is padded.
func overlayRows(base, overlay string, height int) string {
	baseRows := splitToHeight(base, height)
	overRows := splitToHeight(overlay, height)
	out := make([]string, height)
	for i := 0; i < height; i++ {
		b, o := baseRows[i], overRows[i]
		if strings.TrimSpace(stripAnsi(o)) == "" {
			out[i] = b
			continue
		}
		// Cell-level overlay: for every visible cell in o, place it over b.
		out[i] = overlayCells(b, o)
	}
	return strings.Join(out, "\n")
}

func splitToHeight(s string, height int) []string {
	rows := strings.Split(s, "\n")
	if len(rows) < height {
		for len(rows) < height {
			rows = append(rows, "")
		}
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return rows
}

// overlayCells takes two rendered rows (ANSI-styled) and produces a row where
// `over`'s non-space cells replace `base`'s cells at the same visible column.
// Cheap implementation: split both by visible column, replace cell-by-cell.
func overlayCells(base, over string) string {
	// Strip both, walk visible cols, pick `over` where non-space.
	// For perf this is fine — only runs when transforms are active.
	baseRunes := []rune(stripAnsi(base))
	overRunes := []rune(stripAnsi(over))
	maxLen := len(baseRunes)
	if len(overRunes) > maxLen {
		maxLen = len(overRunes)
	}
	var b strings.Builder
	for i := 0; i < maxLen; i++ {
		switch {
		case i < len(overRunes) && overRunes[i] != ' ':
			// Re-style: bold accent for borrowed cells. We're losing the
			// finer per-cell styles from the overlay's ANSI here — fine for
			// the demo; the proper fix is to teach the compositor to emit
			// row-by-row with style preserved.
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render(string(overRunes[i])))
		case i < len(baseRunes):
			b.WriteRune(baseRunes[i])
		default:
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// stripAnsi removes ANSI SGR sequences from a string, leaving raw runes.
// Used by the overlay merge so we can compare visible columns. Kept local
// to lobby for now since the typo package already has one we'd want to
// consolidate to later.
func stripAnsi(s string) string {
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

// scrollbackRows clamps the scroll offset and returns the visible slice as a
// height-length string slice. Empty positions are empty strings so the
// compositor can show the field behind them.
func scrollbackRows(lines []string, height, scroll int) []string {
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := len(lines) - scroll
	start := end - height
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]
	// Bottom-align: pad the TOP with empty rows so chat hugs the bottom of
	// the scrollback area. Those empty rows are where the field shows.
	if len(visible) < height {
		pad := make([]string, height-len(visible))
		visible = append(pad, visible...)
	}
	return visible
}

