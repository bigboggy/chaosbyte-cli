// Package auth wires the GitHub device flow to the identity store. One
// *Service is shared by all SSH sessions on a server; lobbies call it to look
// up known fingerprints at connect time and to run the /auth flow on demand.
//
// New returns nil when GitHub auth isn't configured (no client id). Callers
// must tolerate a nil *Service and treat it as "feature disabled."
package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/bchayka/gitstatus/internal/github"
	"github.com/bchayka/gitstatus/internal/identity"
)

// Service is the facade the lobby talks to.
type Service struct {
	flow  *github.DeviceFlow
	store *identity.Store
}

// New returns a service if clientID is non-empty. store is required when
// returning non-nil (the caller decides how to source it).
func New(clientID string, store *identity.Store) *Service {
	if clientID == "" || store == nil {
		return nil
	}
	return &Service{
		flow: &github.DeviceFlow{
			ClientID: clientID,
			Scope:    "read:user",
			HTTP:     &http.Client{Timeout: 20 * time.Second},
		},
		store: store,
	}
}

// Lookup returns the stored GitHub login for fingerprint, or "" if none.
func (s *Service) Lookup(fingerprint string) string {
	if s == nil {
		return ""
	}
	v, _ := s.store.Lookup(fingerprint)
	return v
}

// StartFlow kicks off a new device authorization. The returned StartResponse
// contains the user code + verification URI to display to the user, plus the
// device code + interval to feed back into PollFlow.
func (s *Service) StartFlow(ctx context.Context) (*github.StartResponse, error) {
	return s.flow.Start(ctx)
}

// PollFlow blocks until the user completes the flow on github.com (returns
// the GitHub login), the code expires, or ctx is cancelled. The access token
// is discarded after resolving the login — we never persist it.
func (s *Service) PollFlow(ctx context.Context, deviceCode string, interval time.Duration) (string, error) {
	token, err := s.flow.Poll(ctx, deviceCode, interval)
	if err != nil {
		return "", err
	}
	return s.flow.UserLogin(ctx, token)
}

// Link persists fingerprint -> ghLogin so future connections by the same SSH
// key are pre-authenticated.
func (s *Service) Link(fingerprint, ghLogin string) error {
	return s.store.Link(fingerprint, ghLogin)
}
