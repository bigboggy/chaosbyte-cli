// Package identity carries the per-user identity model. A Principal is
// the actor identity attached to every event the bus carries, every
// privileged action, and every persisted log entry. It is created
// from a verified SSH ed25519 pubkey at session start and lives for the
// lifetime of the session.
//
// Identity here is intentionally minimal: pubkey-derived ID, display
// name from the allowlist, team membership, session UUID, and a Kind
// discriminator (human or agent). Phase 1 ships only KindHuman; KindAgent
// lands in Phase 4 when each user's session can spawn agents.
package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// PrincipalKind discriminates between a human SSH user and an agent
// spawned by a human's session.
type PrincipalKind int

const (
	KindHuman PrincipalKind = iota
	KindAgent
)

// String returns the canonical wire form for the kind discriminator.
func (k PrincipalKind) String() string {
	switch k {
	case KindAgent:
		return "agent"
	default:
		return "human"
	}
}

// Principal is the actor identity carried on every Event Header and
// every authorized action. Created from a verified SSH pubkey at
// session start; lives for the lifetime of the session. Value type;
// safe to copy.
type Principal struct {
	// ID is the canonical, pubkey-derived identifier. For v0.1:
	// "pk:" + hex(sha256(pubkey)). Forward-compatible with ATProto
	// DIDs at v3 when the prefix becomes "did:plc:..." or similar.
	ID string

	// DisplayName comes from the allowlist entry; falls back to a
	// short fingerprint of the pubkey if no display name is registered.
	DisplayName string

	// PublicKey is the verified ed25519 key. Kept for re-signing
	// capabilities during this session.
	PublicKey ed25519.PublicKey

	// Teams is the list of team slugs this principal is authorized to
	// join.
	Teams []string

	// Roles assigned via the allowlist (v0.1 supports "operator" only;
	// future: "moderator", "admin").
	Roles []string

	// SessionID is unique per SSH connection.
	SessionID uuid.UUID

	// Kind is KindHuman (the SSH-connecting user) or KindAgent (an
	// agent spawned by a session). v0.1 ships only KindHuman.
	Kind PrincipalKind
}

// IDFromPubkey derives the canonical Principal.ID from an ed25519
// pubkey. Deterministic; the same key produces the same ID across
// restarts.
func IDFromPubkey(pk ed25519.PublicKey) string {
	sum := sha256.Sum256(pk)
	return "pk:" + hex.EncodeToString(sum[:])
}

// FingerprintShort returns the first eight hex characters of the
// pubkey hash, useful for log lines and fallback display names.
func FingerprintShort(pk ed25519.PublicKey) string {
	sum := sha256.Sum256(pk)
	return hex.EncodeToString(sum[:4])
}

// IsMemberOf reports whether the principal is authorized to join the
// given team slug.
func (p Principal) IsMemberOf(team string) bool {
	for _, t := range p.Teams {
		if t == team {
			return true
		}
	}
	return false
}

// HasRole reports whether the principal carries the named role.
func (p Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// String is the canonical log form: "ID/kind/session".
func (p Principal) String() string {
	return p.ID + "/" + p.Kind.String() + "/" + p.SessionID.String()
}
