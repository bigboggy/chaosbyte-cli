package events

import (
	"encoding/binary"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/glycerine/hlc"
)

// Timestamp is the canonical HLC stamp used on every Event Header.
// It wraps github.com/glycerine/hlc.HLC (an int64 under the hood) so
// the events package owns the serialization shape and the broker uses
// our naming conventions instead of the library's.
type Timestamp struct {
	hlc.HLC
}

// IsZero reports whether the stamp has been assigned.
func (t Timestamp) IsZero() bool { return int64(t.HLC) == 0 }

// Before reports whether t is strictly earlier than other.
func (t Timestamp) Before(other Timestamp) bool { return int64(t.HLC) < int64(other.HLC) }

// After reports whether t is strictly later than other.
func (t Timestamp) After(other Timestamp) bool { return int64(t.HLC) > int64(other.HLC) }

// MarshalJSON encodes the stamp as a JSON number so it survives
// json.Marshal without quotes (8 bytes of int64 in decimal).
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(t.HLC))
}

// UnmarshalJSON decodes a JSON number back into the stamp.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	t.HLC = hlc.HLC(v)
	return nil
}

// Bytes returns the 8-byte big-endian wire form. Used by the persistent
// store to write fixed-width columns.
func (t Timestamp) Bytes() []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(t.HLC))
	return b[:]
}

// FromBytes reconstructs a Timestamp from the 8-byte wire form.
func FromBytes(b []byte) (Timestamp, error) {
	if len(b) != 8 {
		return Timestamp{}, ErrInvalidTimestamp
	}
	return Timestamp{HLC: hlc.HLC(binary.BigEndian.Uint64(b))}, nil
}

// ToTime returns the wall-clock portion as a Go time.Time. Useful for
// rendering "X seconds ago" in the UI.
func (t Timestamp) ToTime() time.Time {
	return t.HLC.ToTime()
}

// ErrInvalidTimestamp is returned by FromBytes if the input length is
// wrong.
var ErrInvalidTimestamp = jsonError("invalid timestamp wire format")

// jsonError is a sentinel error type kept local so callers can type
// switch without importing this package's errors.
type jsonError string

func (e jsonError) Error() string { return string(e) }

// Clock wraps the underlying hlc.HLC with the naming the design docs
// use ("Now", "Update") and goroutine-safe accessors. Internally it is
// a single int64; mutating methods use atomic operations.
type Clock struct {
	state atomic.Int64
}

// NewClock returns a fresh clock whose state will be initialized to the
// first call to Now(). Optionally seeds from a known prior stamp so
// daemon restarts can advance past durable state.
func NewClock(seed ...Timestamp) *Clock {
	c := &Clock{}
	if len(seed) > 0 {
		c.state.Store(int64(seed[0].HLC))
	}
	return c
}

// Now emits a new local timestamp, advancing the clock as needed.
func (c *Clock) Now() Timestamp {
	for {
		current := hlc.HLC(c.state.Load())
		next := current
		stamp := next.CreateSendOrLocalEvent()
		if c.state.CompareAndSwap(int64(current), int64(next)) {
			return Timestamp{HLC: stamp}
		}
	}
}

// Update merges a remote stamp into the local clock. The local clock
// advances such that subsequent Now() calls produce stamps strictly
// later than both the prior local state and the remote.
func (c *Clock) Update(remote Timestamp) Timestamp {
	for {
		current := hlc.HLC(c.state.Load())
		next := current
		merged := next.ReceiveMessageWithHLC(remote.HLC)
		if c.state.CompareAndSwap(int64(current), int64(next)) {
			return Timestamp{HLC: merged}
		}
	}
}
