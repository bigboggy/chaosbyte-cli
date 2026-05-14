// Package capability wraps biscuit-go/v2 to issue and verify capability
// tokens for the vibespace and monobyte event bus. A token binds a
// principal to a set of room scopes, action classes, and an expiration;
// the broker verifies the token on each Publish before fan-out.
//
// Phase 1 ships the scaffold. The broker's verifier passes nil tokens
// through (CapabilityProof is nil pre-Phase 5); IssueSession is
// available for the cmd/vibespace-server handler to mint a session
// token when pubkey auth succeeds. Phase 5 turns on per-event
// verification when MOACP intents start flowing.
package capability

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	bisc "github.com/biscuit-auth/biscuit-go/v2"
	"github.com/biscuit-auth/biscuit-go/v2/parser"

	"github.com/bchayka/gitstatus/internal/identity"
)

// Issuer holds the daemon's biscuit root key and mints session tokens.
// One Issuer per daemon; load or generate on startup.
type Issuer struct {
	rootPub  ed25519.PublicKey
	rootPriv ed25519.PrivateKey
}

// NewIssuer loads the biscuit root key from path. If the file does
// not exist, generates a new keypair and writes it. The private key
// file is 0600-permissioned.
func NewIssuer(path string) (*Issuer, error) {
	priv, pub, err := loadOrGenerateRootKey(path)
	if err != nil {
		return nil, err
	}
	return &Issuer{rootPub: pub, rootPriv: priv}, nil
}

// IssueSession mints a biscuit token scoped to the principal's teams
// and roles. The token expires in ttl (typically 1 hour for a session).
// Returns the serialized token; the caller passes it through to the
// event Header's CapabilityProof field.
func (i *Issuer) IssueSession(p identity.Principal, ttl time.Duration) (Token, error) {
	expiry := time.Now().Add(ttl).UTC().Format(time.RFC3339)
	teamFacts := ""
	for _, t := range p.Teams {
		teamFacts += fmt.Sprintf("team(%q);\n", t)
	}
	roleFacts := ""
	for _, r := range p.Roles {
		roleFacts += fmt.Sprintf("role(%q);\n", r)
	}

	source := fmt.Sprintf(`
		user(%q);
		display(%q);
		%s%s
		expiry(%q);
		check if room($r), team($r);
	`, p.ID, p.DisplayName, teamFacts, roleFacts, expiry)

	block, err := parser.FromStringBlockWithParams(source, nil)
	if err != nil {
		return Token{}, fmt.Errorf("capability: build authority block: %w", err)
	}

	builder := bisc.NewBuilder(i.rootPriv)
	if err := builder.AddBlock(block); err != nil {
		return Token{}, fmt.Errorf("capability: add block: %w", err)
	}
	b, err := builder.Build()
	if err != nil {
		return Token{}, fmt.Errorf("capability: build: %w", err)
	}
	raw, err := b.Serialize()
	if err != nil {
		return Token{}, fmt.Errorf("capability: serialize: %w", err)
	}
	return Token{raw: raw, parsed: b}, nil
}

// RootPublicKey returns the issuer's public key. Verifiers use this to
// authorize tokens; safe to embed in configs or expose over the
// network.
func (i *Issuer) RootPublicKey() ed25519.PublicKey { return i.rootPub }

// Token wraps a serialized biscuit. Construct via Issuer.IssueSession
// or capability.ParseToken.
type Token struct {
	raw    []byte
	parsed *bisc.Biscuit
}

// Bytes returns the raw serialized biscuit. Suitable for setting on
// events.Header.CapabilityProof.
func (t Token) Bytes() []byte { return t.raw }

// IsEmpty reports whether the token holds no data (the zero value).
func (t Token) IsEmpty() bool { return len(t.raw) == 0 }

// ParseToken decodes a serialized biscuit without verifying it. Useful
// for tools that want to inspect tokens without holding the verifier
// key.
func ParseToken(raw []byte) (Token, error) {
	if len(raw) == 0 {
		return Token{}, errors.New("capability: empty token")
	}
	b, err := bisc.Unmarshal(raw)
	if err != nil {
		return Token{}, fmt.Errorf("capability: unmarshal: %w", err)
	}
	return Token{raw: raw, parsed: b}, nil
}

// loadOrGenerateRootKey reads the ed25519 private key from path or
// generates and persists a new one. The file format is the 32-byte
// seed followed by the 32-byte public key (the standard ed25519
// PrivateKey layout).
func loadOrGenerateRootKey(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if b, err := os.ReadFile(path); err == nil {
		if len(b) != ed25519.PrivateKeySize {
			return nil, nil, fmt.Errorf("capability: %s has wrong key size %d", path, len(b))
		}
		priv := ed25519.PrivateKey(b)
		return priv, priv.Public().(ed25519.PublicKey), nil
	} else if !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("capability: read %s: %w", path, err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("capability: generate key: %w", err)
	}
	if err := os.WriteFile(path, priv, 0o600); err != nil {
		return nil, nil, fmt.Errorf("capability: write %s: %w", path, err)
	}
	return priv, pub, nil
}
