package identity

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/ssh"
)

// Allowlist holds the set of authorized principals keyed by their
// pubkey. Goroutine-safe; Reload swaps the entire set atomically.
type Allowlist struct {
	mu      sync.RWMutex
	entries map[string]allowlistEntry // keyed by hex(sha256(pubkey))
	path    string
}

type allowlistEntry struct {
	Display string
	PubKey  ed25519.PublicKey
	Teams   []string
	Roles   []string
}

// tomlFile mirrors the on-disk TOML schema.
type tomlFile struct {
	Principals []tomlPrincipal `toml:"principals"`
}

type tomlPrincipal struct {
	Display string   `toml:"display"`
	Pubkey  string   `toml:"pubkey"` // either an OpenSSH-format line or raw hex
	Teams   []string `toml:"teams"`
	Roles   []string `toml:"roles"`
}

// LoadAllowlist reads the TOML allowlist from disk. Returns an error
// if the file is missing or unparseable.
func LoadAllowlist(path string) (*Allowlist, error) {
	a := &Allowlist{
		entries: map[string]allowlistEntry{},
		path:    path,
	}
	if err := a.Reload(path); err != nil {
		return nil, err
	}
	return a, nil
}

// Reload re-reads the allowlist from the path it was loaded from (or a
// new path). Atomic swap; readers in flight see either the old set or
// the new, never a partial state.
func (a *Allowlist) Reload(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("allowlist: read %s: %w", path, err)
	}
	var raw tomlFile
	if err := toml.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("allowlist: parse %s: %w", path, err)
	}
	next := map[string]allowlistEntry{}
	for _, p := range raw.Principals {
		if p.Pubkey == "" || p.Display == "" {
			continue // skip malformed entries
		}
		pk, err := parsePubkey(p.Pubkey)
		if err != nil {
			// Skip but do not fail; log via stdlib so an unreadable
			// entry does not knock the daemon offline.
			fmt.Fprintf(os.Stderr, "allowlist: skip %q: %v\n", p.Display, err)
			continue
		}
		key := IDFromPubkey(pk)
		next[key] = allowlistEntry{
			Display: p.Display,
			PubKey:  pk,
			Teams:   p.Teams,
			Roles:   p.Roles,
		}
	}
	a.mu.Lock()
	a.entries = next
	a.path = path
	a.mu.Unlock()
	return nil
}

// Lookup returns the allowlist entry for a pubkey, or false if it is
// not authorized.
func (a *Allowlist) Lookup(pk ed25519.PublicKey) (allowlistEntry, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	e, ok := a.entries[IDFromPubkey(pk)]
	return e, ok
}

// Count returns the number of entries currently loaded.
func (a *Allowlist) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.entries)
}

// parsePubkey accepts an OpenSSH-format pubkey line ("ssh-ed25519 AAAA...
// comment") and returns the underlying ed25519 public key. Errors if the
// key type is not ed25519.
func parsePubkey(s string) (ed25519.PublicKey, error) {
	out, _, _, _, err := ssh.ParseAuthorizedKey([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("parse authorized key: %w", err)
	}
	// x/crypto/ssh's ParseAuthorizedKey returns an ssh.PublicKey; for
	// ed25519 we cryptoPublicKey-cast through its CryptoPublicKey
	// interface if the implementation provides it.
	if cp, ok := out.(ssh.CryptoPublicKey); ok {
		if ed, ok := cp.CryptoPublicKey().(ed25519.PublicKey); ok {
			return ed, nil
		}
	}
	return nil, errors.New("not an ed25519 public key")
}
