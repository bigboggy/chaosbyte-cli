// Package identity persists the mapping from an SSH public key fingerprint to
// a GitHub login. Once a user runs `/auth github` and proves they own a
// GitHub account, the (fingerprint, login) pair is stored here so future
// connections by the same SSH key skip the auth flow.
//
// The format is a flat JSON object on disk:
//
//	{
//	  "SHA256:abcdef...": "octocat",
//	  "SHA256:fedcba...": "torvalds"
//	}
//
// Storage intentionally does NOT include access tokens — only the public
// mapping is sensitive (it associates a public key with a public username),
// not a credential.
package identity

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Store is the on-disk identity map. All methods are safe for concurrent use.
type Store struct {
	mu   sync.Mutex
	path string
	data map[string]string
}

// Open loads the store at path. If path does not exist, an empty store is
// returned and the file will be created on the first Link.
func Open(path string) (*Store, error) {
	s := &Store{path: path, data: make(map[string]string)}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read identity store: %w", err)
	}
	if len(b) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("parse identity store: %w", err)
	}
	return s, nil
}

// Lookup returns the GitHub login linked to a fingerprint.
func (s *Store) Lookup(fingerprint string) (string, bool) {
	if fingerprint == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[fingerprint]
	return v, ok
}

// Unlink removes the mapping for fingerprint and persists. No-op if the
// fingerprint wasn't linked.
func (s *Store) Unlink(fingerprint string) error {
	if fingerprint == "" {
		return errors.New("empty fingerprint")
	}
	s.mu.Lock()
	if _, ok := s.data[fingerprint]; !ok {
		s.mu.Unlock()
		return nil
	}
	delete(s.data, fingerprint)
	clone := make(map[string]string, len(s.data))
	for k, v := range s.data {
		clone[k] = v
	}
	s.mu.Unlock()
	return s.writeAtomic(clone)
}

// Link stores fingerprint -> ghLogin and persists. Overwrites any prior
// mapping for the same fingerprint.
func (s *Store) Link(fingerprint, ghLogin string) error {
	if fingerprint == "" {
		return errors.New("empty fingerprint")
	}
	if ghLogin == "" {
		return errors.New("empty github login")
	}
	s.mu.Lock()
	s.data[fingerprint] = ghLogin
	clone := make(map[string]string, len(s.data))
	for k, v := range s.data {
		clone[k] = v
	}
	s.mu.Unlock()

	return s.writeAtomic(clone)
}

// writeAtomic serializes data to a temp file then renames it over the target,
// so a crash mid-write can't corrupt the store.
func (s *Store) writeAtomic(data map[string]string) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir identity dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "identity-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("encode: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
