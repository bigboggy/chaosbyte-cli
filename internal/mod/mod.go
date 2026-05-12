// Package mod is the rules-v0 chatroom moderator. It posts idle prompts,
// periodic suggestions, and welcomes on join. The same surface is what the
// LLM-driven mod will plug into later: Tick advances the clock, NoteChat
// resets the idle timer, and the Mod returns lines for the host to post.
package mod

import (
	"math/rand"
	"strings"
	"time"
)

const (
	// PromptInterval is the longest gap between mod nudges, in seconds.
	PromptInterval = 90 * time.Second
	// IdleThreshold says how quiet the room has to be before the mod fills
	// the gap with an unprompted line.
	IdleThreshold = 45 * time.Second
)

// Mod is the rules-v0 chatroom moderator. Hosts feed it ticks and notifications
// of activity; it returns lines to post.
type Mod struct {
	lastChat   time.Time
	nextPrompt time.Time
	rng        *rand.Rand

	prompts []string
}

// New returns a fresh moderator armed for its first prompt one interval out.
func New() *Mod {
	now := time.Now()
	return &Mod{
		lastChat:   now,
		nextPrompt: now.Add(PromptInterval),
		rng:        rand.New(rand.NewSource(now.UnixNano())),
		prompts:    defaultPrompts(),
	}
}

// defaultPrompts is the canned line pool the rules-v0 mod draws from. The
// LLM-driven mod will generate these on demand once the wiring is in.
func defaultPrompts() []string {
	return []string{
		"spotlight rotates every minute · /spotlight to dive in",
		"slow afternoon? /games has bricks blitz · 30s of paddle physics",
		"news feed is fresh today · /news",
		"discussions has a take that needs a counter-take · /discussions",
		"resources lists are sortable · /resources",
		"type a slash to pull up the command palette",
		"ambient room is just the engine talking to itself · /ambient",
	}
}

// Welcome returns the mod's greeting for a newly-joined nick.
func (m *Mod) Welcome(nick string) string {
	return "welcome, " + nick + ". the workshop is yours for as long as you like."
}

// NoteChat resets the idle clock — call this whenever a user or system
// posts a regular chat line. Without it the mod thinks the room is dead
// and fills every gap.
func (m *Mod) NoteChat(t time.Time) {
	m.lastChat = t
}

// Tick advances the mod's internal clock. If a prompt is due, returns the
// body the host should post; otherwise returns "". Two rules fire:
//   - scheduled: every PromptInterval since the last prompt, the mod fires
//   - idle gap: if no chat for IdleThreshold and at least 15s since the last
//     prompt, the mod fires to break the silence
func (m *Mod) Tick(t time.Time) string {
	dueScheduled := !t.Before(m.nextPrompt)
	dueIdle := t.Sub(m.lastChat) > IdleThreshold &&
		t.Sub(m.nextPrompt.Add(-PromptInterval)) > 15*time.Second
	if !dueScheduled && !dueIdle {
		return ""
	}
	m.nextPrompt = t.Add(PromptInterval)
	if len(m.prompts) == 0 {
		return ""
	}
	return m.prompts[m.rng.Intn(len(m.prompts))]
}

// Nick is the author label the lobby uses when posting mod lines.
const Nick = "@mod"

// QuestionTag returns a tag if body reads as a question, otherwise nil.
// The rule is the rules-v0 heuristic: trim, must be a normal chat line,
// must end in '?' after trimming. Future rules go alongside this one in
// internal/mod and the broker invokes each on every publish.
func QuestionTag(body string) *Tag {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil
	}
	if !strings.HasSuffix(trimmed, "?") {
		return nil
	}
	return &Tag{
		Kind:   "question",
		Marker: '✦',
		Reason: "looks like a question, marked for attention",
		BornAt: time.Now(),
	}
}

// Tag is the moderator's annotation about a chat message. The broker
// attaches Tags to outgoing messages so subscribers receive the message
// already marked. Tag mirrors ui.ChatTag without depending on the ui
// package, which would create a cycle.
type Tag struct {
	Kind   string
	Marker rune
	Reason string
	BornAt time.Time
}
