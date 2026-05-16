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

	"github.com/bigboggy/vibespace/internal/github"
	"github.com/bigboggy/vibespace/internal/identity"
	"github.com/bigboggy/vibespace/internal/store"
)

// Service is the facade the lobby talks to.
type Service struct {
	flow     *github.DeviceFlow
	identity *identity.Store
	data     *store.Store // profile cache; may be nil
}

// New returns a service if clientID is non-empty. identityStore is required
// when returning non-nil. dataStore is optional — when set, /auth caches the
// user's GitHub profile (repos, stars, contributions) alongside the link so
// future profile views don't need a live API call.
func New(clientID string, identityStore *identity.Store, dataStore *store.Store) *Service {
	if clientID == "" || identityStore == nil {
		return nil
	}
	return &Service{
		flow: &github.DeviceFlow{
			ClientID: clientID,
			// read:user covers /user, public repos, and the GraphQL
			// contributionsCollection. We also need it for /user/starred.
			Scope: "read:user",
			HTTP:  &http.Client{Timeout: 30 * time.Second},
		},
		identity: identityStore,
		data:     dataStore,
	}
}

// Lookup returns the stored GitHub login for fingerprint, or "" if none.
func (s *Service) Lookup(fingerprint string) string {
	if s == nil {
		return ""
	}
	v, _ := s.identity.Lookup(fingerprint)
	return v
}

// StartFlow kicks off a new device authorization. The returned StartResponse
// contains the user code + verification URI to display to the user, plus the
// device code + interval to feed back into PollFlow.
func (s *Service) StartFlow(ctx context.Context) (*github.StartResponse, error) {
	return s.flow.Start(ctx)
}

// PollFlow blocks until the user completes the flow on github.com, the code
// expires, or ctx is cancelled. On success it resolves the user's profile
// (login, repos, stars, contributions) and caches it in the data store, then
// returns the login. The access token is discarded after this call returns —
// we never persist it.
func (s *Service) PollFlow(ctx context.Context, deviceCode string, interval time.Duration) (string, error) {
	token, err := s.flow.Poll(ctx, deviceCode, interval)
	if err != nil {
		return "", err
	}
	if s.data == nil {
		// No data store configured — fall back to just resolving the login.
		return s.flow.UserLogin(ctx, token)
	}
	p, err := s.flow.FetchProfile(ctx, token)
	if err != nil {
		return "", err
	}
	if err := s.cacheProfile(p); err != nil {
		// Cache failure shouldn't block login — surface but continue.
		return p.Login, nil
	}
	return p.Login, nil
}

// cacheProfile writes a fetched GitHub profile into the data store. Each
// section is best-effort; a single failure doesn't poison the others.
func (s *Service) cacheProfile(p *github.Profile) error {
	if err := s.data.UpsertUser(store.User{
		Login:       p.Login,
		Name:        p.Name,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		Location:    p.Location,
		Company:     p.Company,
		Followers:   p.Followers,
		Following:   p.Following,
		PublicRepos: p.PublicRepos,
		SyncedAt:    time.Now(),
	}); err != nil {
		return err
	}
	repos := make([]store.Repo, 0, len(p.Repos))
	for _, r := range p.Repos {
		repos = append(repos, store.Repo{
			Name:        r.Name,
			Description: r.Description,
			Language:    r.Language,
			Stars:       r.Stars,
			Forks:       r.Forks,
			IsFork:      r.IsFork,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	_ = s.data.ReplaceRepos(p.Login, repos)

	stars := make([]store.StarredRepo, 0, len(p.Stars))
	for _, r := range p.Stars {
		stars = append(stars, store.StarredRepo{
			FullName:    r.FullName,
			Description: r.Description,
			Language:    r.Language,
			Stars:       r.Stars,
		})
	}
	_ = s.data.ReplaceStars(p.Login, stars)

	days := make([]store.ContribDay, 0, len(p.Contributions))
	for _, d := range p.Contributions {
		days = append(days, store.ContribDay{Date: d.Date, Count: d.Count})
	}
	_ = s.data.ReplaceContributions(p.Login, days)
	return nil
}

// Link persists fingerprint -> ghLogin so future connections by the same SSH
// key are pre-authenticated.
func (s *Service) Link(fingerprint, ghLogin string) error {
	return s.identity.Link(fingerprint, ghLogin)
}

// Unlink removes the stored mapping for fingerprint.
func (s *Service) Unlink(fingerprint string) error {
	return s.identity.Unlink(fingerprint)
}
