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
// each get their own buffered channel.
type Broker struct {
	mu       sync.Mutex
	messages map[string][]ui.ChatMessage
	subs     map[string][]chan Event
	mod      *mod.Mod
	stop     chan struct{}
}

// New starts a broker with #lobby pre-seeded and a mod goroutine running.
func New() *Broker {
	b := &Broker{
		messages: map[string][]ui.ChatMessage{},
		subs:     map[string][]chan Event{},
		mod:      mod.New(),
		stop:     make(chan struct{}),
	}
	for ch, msgs := range seedMessages() {
		b.messages[ch] = msgs
	}
	go b.runMod()
	return b
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

// Subscribe returns a channel that receives every future Event for the
// named room channel. Send is non-blocking; if the subscriber is slow the
// event is dropped.
func (b *Broker) Subscribe(channel string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := make(chan Event, 32)
	b.subs[channel] = append(b.subs[channel], sub)
	return sub
}

// Publish appends a message to the channel and broadcasts to subscribers.
// The mod's idle clock is reset on every publish. A ChatJoin kind triggers
// a follow-up mod welcome so other sessions see "@mod welcomes @newnick".
func (b *Broker) Publish(channel string, msg ui.ChatMessage) {
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
	subs := append([]chan Event(nil), b.subs[channel]...)
	b.mu.Unlock()
	for _, s := range subs {
		select {
		case s <- Event{Channel: channel, Message: msg}:
		default:
		}
	}
}
