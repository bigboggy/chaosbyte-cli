// Package room is the shared multi-user state — chat messages broadcast
// across sessions. Today it owns one channel (#lobby) and a single mod that
// fires for the whole room; per-channel topic/online lists stay on the
// lobby Screen for now.
package room

import (
	"sync"
	"time"

	"github.com/bchayka/gitstatus/internal/mod"
	"github.com/bchayka/gitstatus/internal/ui"
)

// Channel is the shared chat scrollback for a room channel.
type Channel struct {
	Name string
}

// Event fires for every message published to a channel.
type Event struct {
	Channel string
	Message ui.ChatMessage
}

// Broker is the shared room state. It's safe for concurrent use; subscribers
// each get their own buffered channel and receive events for every channel
// the broker hosts.
type Broker struct {
	mu       sync.Mutex
	messages map[string][]ui.ChatMessage
	subs     []chan Event
	mod      *mod.Mod
	stop     chan struct{}

	// publish timestamps within the last activityWindow seconds. Drives
	// Tier() so every screen can read the room's energy level without
	// computing its own.
	pubTimes []time.Time
}

// activityWindow is the lookback the tier classifier uses on publish times.
const activityWindow = 10 * time.Second

// New starts a broker with every seeded channel ready and a mod goroutine
// running. Today the seeds live in room.seedMessages; future builds will
// pull from persistent storage.
func New() *Broker {
	b := &Broker{
		messages: map[string][]ui.ChatMessage{},
		mod:      mod.New(),
		stop:     make(chan struct{}),
	}
	for ch, msgs := range seedMessages() {
		b.messages[ch] = msgs
	}
	go b.runMod()
	return b
}

// Channels returns the list of seeded channel names. Stable across the
// broker's lifetime (no dynamic create yet).
func (b *Broker) Channels() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.messages))
	for name := range b.messages {
		out = append(out, name)
	}
	return out
}

// Stop terminates the broker's mod goroutine. Subscribers stay open; this
// is for orderly server shutdown.
func (b *Broker) Stop() {
	select {
	case <-b.stop:
	default:
		close(b.stop)
	}
}

// runMod ticks the moderator at 1Hz and publishes any line it returns.
// One mod per room; per-session mods would duplicate prompts.
func (b *Broker) runMod() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-b.stop:
			return
		case t := <-ticker.C:
			if line := b.mod.Tick(t); line != "" {
				b.Publish("#lobby", ui.ChatMessage{
					Author: mod.Nick, Body: line, At: t, Kind: ui.ChatAction,
				})
			}
		}
	}
}

// Snapshot returns a copy of the channel's scrollback. Callers can iterate
// without holding the broker lock.
func (b *Broker) Snapshot(channel string) []ui.ChatMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	src := b.messages[channel]
	out := make([]ui.ChatMessage, len(src))
	copy(out, src)
	return out
}

// Subscribe returns a channel that receives every future Event the broker
// publishes, regardless of channel. The subscriber inspects evt.Channel to
// route. Send is non-blocking; if the subscriber is slow the event drops.
func (b *Broker) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := make(chan Event, 64)
	b.subs = append(b.subs, sub)
	return sub
}

// Publish appends a message to the channel and broadcasts to subscribers.
// The mod's idle clock is reset on every publish. A ChatJoin kind triggers
// a follow-up mod welcome so other sessions see "@mod welcomes @newnick".
//
// Before broadcast the moderator inspects the message and may attach tags.
// The rules-v0 detector handles questions (lines that end with '?'); future
// rules cover URLs, code, builds, and repo drops. Tagged messages carry
// the moderator's marker in the margin once they arrive in subscribers'
// views.
func (b *Broker) Publish(channel string, msg ui.ChatMessage) {
	if msg.Kind == ui.ChatNormal {
		if tag := mod.QuestionTag(msg.Body); tag != nil {
			msg.Tags = append(msg.Tags, ui.ChatTag{
				Kind:   tag.Kind,
				Marker: tag.Marker,
				Reason: tag.Reason,
				BornAt: tag.BornAt,
			})
		}
	}
	b.publish(channel, msg)
	if msg.Kind == ui.ChatJoin && msg.Author != "" {
		b.publish(channel, ui.ChatMessage{
			Author: mod.Nick,
			Body:   b.mod.Welcome(msg.Author),
			At:     msg.At,
			Kind:   ui.ChatAction,
		})
	}
}

func (b *Broker) publish(channel string, msg ui.ChatMessage) {
	b.mu.Lock()
	b.messages[channel] = append(b.messages[channel], msg)
	b.mod.NoteChat(msg.At)
	b.pubTimes = append(b.pubTimes, msg.At)
	b.trimPubTimes(msg.At)
	subs := append([]chan Event(nil), b.subs...)
	b.mu.Unlock()
	for _, s := range subs {
		select {
		case s <- Event{Channel: channel, Message: msg}:
		default:
		}
	}
}

// trimPubTimes drops timestamps older than activityWindow. Caller holds mu.
func (b *Broker) trimPubTimes(now time.Time) {
	cutoff := now.Add(-activityWindow)
	keep := b.pubTimes[:0]
	for _, t := range b.pubTimes {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	b.pubTimes = keep
}

// Tier returns the room's current intensity tier derived from publish
// frequency in the last activityWindow seconds. Screens read this each
// tick to keep the field's energy aligned with the room:
//   - 0 quiet: no activity in window
//   - 1 reactive: 1-2 events
//   - 2 eventful: 3-5 events
//   - 3 hype: 6+ events
//
// Tier 4 (game takeover) is screen-local, not room-wide.
func (b *Broker) Tier() int {
	b.mu.Lock()
	b.trimPubTimes(time.Now())
	n := len(b.pubTimes)
	b.mu.Unlock()
	switch {
	case n >= 6:
		return 3
	case n >= 3:
		return 2
	case n >= 1:
		return 1
	}
	return 0
}
