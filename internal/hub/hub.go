// Package hub is the server-side source of truth for channels and messages.
//
// One Hub is shared across all SSH sessions. Each session subscribes to receive
// Events when state changes; the session's bubbletea program re-renders by
// reading back from the Hub. The Hub is the only place that holds mutable chat
// state — sessions own only their own UI state (input, scroll, history).
package hub

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// Event signals "something changed in the hub" to subscribers. The kind/name
// pair is a hint; subscribers re-read the hub to get the new state rather than
// trusting the event payload, which keeps the wire shape trivial.
//
// Event also satisfies tea.Msg directly, so it can be dispatched through a
// bubbletea program without per-screen wrappers.
type Event struct {
	Kind    EventKind
	Channel string // affected channel name; empty for hub-wide events
}

type EventKind int

const (
	// EventMessage — a new message landed in Channel.
	EventMessage EventKind = iota
	// EventChannelCreated — a new channel exists.
	EventChannelCreated
	// EventPresence — someone joined/left a channel, or connected/disconnected.
	EventPresence
)

// Channel is a single chat room. Messages are append-only.
type Channel struct {
	Name     string
	Messages []ui.ChatMessage
}

// Hub owns the channel list and broadcasts changes to subscribers.
type Hub struct {
	mu       sync.RWMutex
	order    []string            // channel names in display order; "#lobby" first
	channels map[string]*Channel // name -> channel
	subs     map[uint64]*sub     // id -> subscriber

	// viewing maps subscriber id -> the channel they're currently looking at.
	// Used to compute per-channel online counts.
	viewing map[uint64]string

	nextSub atomic.Uint64
}

type sub struct {
	events chan Event
	closed atomic.Bool
}

// New returns a Hub seeded with the default set of channels and lobby MOTD.
func New() *Hub {
	h := &Hub{
		channels: make(map[string]*Channel),
		subs:     make(map[uint64]*sub),
		viewing:  make(map[uint64]string),
	}
	for _, c := range seed() {
		h.channels[c.Name] = c
		h.order = append(h.order, c.Name)
	}
	return h
}

// ChannelNames returns channel names in display order (a snapshot).
func (h *Hub) ChannelNames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, len(h.order))
	copy(out, h.order)
	return out
}

// Messages returns a snapshot of the messages in the named channel. Returns
// (nil, false) if the channel doesn't exist.
func (h *Hub) Messages(name string) ([]ui.ChatMessage, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ch, ok := h.channels[name]
	if !ok {
		return nil, false
	}
	out := make([]ui.ChatMessage, len(ch.Messages))
	copy(out, ch.Messages)
	return out, true
}

// HasChannel reports whether a channel with the given name exists.
func (h *Hub) HasChannel(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.channels[name]
	return ok
}

// Online returns the number of subscribers currently viewing the named
// channel.
func (h *Hub) Online(name string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, v := range h.viewing {
		if v == name {
			n++
		}
	}
	return n
}

// Post appends a message to the named channel and broadcasts. If the channel
// doesn't exist, this is a no-op.
func (h *Hub) Post(channel, author, body string, kind ui.ChatKind) {
	h.mu.Lock()
	ch, ok := h.channels[channel]
	if !ok {
		h.mu.Unlock()
		return
	}
	ch.Messages = append(ch.Messages, ui.ChatMessage{
		Author: author, Body: body, At: time.Now(), Kind: kind,
	})
	h.mu.Unlock()
	h.broadcast(Event{Kind: EventMessage, Channel: channel})
}

// CreateChannel adds a channel if missing. Returns true if it created one.
func (h *Hub) CreateChannel(name string) bool {
	if !strings.HasPrefix(name, "#") {
		name = "#" + name
	}
	h.mu.Lock()
	if _, exists := h.channels[name]; exists {
		h.mu.Unlock()
		return false
	}
	h.channels[name] = &Channel{Name: name}
	h.order = append(h.order, name)
	h.mu.Unlock()
	h.broadcast(Event{Kind: EventChannelCreated, Channel: name})
	return true
}

// Subscribe registers a subscriber and returns its id and an event channel.
// The channel is buffered; if a subscriber's buffer is full, an event is
// dropped for that subscriber (subscribers re-read state on every event, so
// dropping is non-fatal — only ordering of notifications is affected).
//
// Call Unsubscribe to release resources when the session ends.
func (h *Hub) Subscribe() (uint64, <-chan Event) {
	id := h.nextSub.Add(1)
	s := &sub{events: make(chan Event, 16)}
	h.mu.Lock()
	h.subs[id] = s
	h.mu.Unlock()
	return id, s.events
}

// Unsubscribe drops the subscriber and any presence it had registered.
func (h *Hub) Unsubscribe(id uint64) {
	h.mu.Lock()
	s, ok := h.subs[id]
	if ok {
		delete(h.subs, id)
	}
	prev, viewed := h.viewing[id]
	delete(h.viewing, id)
	h.mu.Unlock()

	if ok && s.closed.CompareAndSwap(false, true) {
		close(s.events)
	}
	if viewed {
		h.broadcast(Event{Kind: EventPresence, Channel: prev})
	}
}

// SetViewing records which channel a subscriber is currently looking at, used
// by Online(). Pass an empty string to clear.
func (h *Hub) SetViewing(id uint64, channel string) {
	h.mu.Lock()
	prev := h.viewing[id]
	if channel == "" {
		delete(h.viewing, id)
	} else {
		h.viewing[id] = channel
	}
	h.mu.Unlock()

	if prev != "" && prev != channel {
		h.broadcast(Event{Kind: EventPresence, Channel: prev})
	}
	if channel != "" && channel != prev {
		h.broadcast(Event{Kind: EventPresence, Channel: channel})
	}
}

// broadcast fans an event out to every subscriber. Drops on full buffer.
func (h *Hub) broadcast(ev Event) {
	h.mu.RLock()
	subs := make([]*sub, 0, len(h.subs))
	for _, s := range h.subs {
		subs = append(subs, s)
	}
	h.mu.RUnlock()

	for _, s := range subs {
		if s.closed.Load() {
			continue
		}
		select {
		case s.events <- ev:
		default:
			// Subscriber is slow; skip rather than block the hub.
		}
	}
}
