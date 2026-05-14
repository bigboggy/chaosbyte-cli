// Package platform is the multi-tenant layer. The same engine renders any
// team's room; the platform decides which team's config and broker a given
// SSH session connects to.
//
// The flagship Vibespace registers itself at the slug "vibespace" with the
// DefaultVibespace config. Other teams register their own config under
// their own slug. An SSH session connects with `ssh teamslug@host`, the
// platform reads the user, resolves to that team's config and broker, and
// the engine renders the room.
//
// Per-team isolation:
//   - Each team has its own RoomConfig (brand, theme, moderator personality)
//   - Each team has its own broker (chat history, member list, spotlight)
//   - Cross-team traffic is impossible by construction
//
// Resolution falls back to the flagship if an unknown slug arrives, so a
// random SSH connection lands in Vibespace rather than being refused.
package platform

import (
	"sync"

	"github.com/bchayka/gitstatus/internal/capability"
	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/room"
)

// Registry holds the active set of teams and routes incoming connections
// to the right one. Safe for concurrent use.
type Registry struct {
	mu           sync.RWMutex
	flagshipSlug string
	configs      map[string]config.RoomConfig
	brokers      map[string]*room.Broker
	verifier     *capability.Issuer
}

// NewRegistry builds a registry seeded with the flagship Vibespace. The
// flagship's slug is whatever DefaultVibespace returns ("vibespace" today).
// Use Register to add more teams. verifier is optional; when set, every
// per-team broker uses it for capability checks.
func NewRegistry(verifier *capability.Issuer) *Registry {
	r := &Registry{
		configs:  map[string]config.RoomConfig{},
		brokers:  map[string]*room.Broker{},
		verifier: verifier,
	}
	flagship := config.DefaultVibespace()
	r.flagshipSlug = flagship.Slug
	r.Register(flagship)
	return r
}

// Register adds a team or replaces an existing one. If the team is new, a
// broker is spun up for it. Re-registering an existing team keeps its
// broker alive so connected users do not see their room reset.
func (r *Registry) Register(cfg config.RoomConfig) {
	cfg = config.MergeWithDefaults(cfg)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[cfg.Slug] = cfg
	if _, ok := r.brokers[cfg.Slug]; !ok {
		r.brokers[cfg.Slug] = room.New(cfg.Slug, nil, r.verifier)
	}
}

// Resolve maps an SSH user to the right team. Unknown slugs land on the
// flagship so an arrival who types the host without thinking still ends
// up somewhere.
func (r *Registry) Resolve(slug string) (config.RoomConfig, *room.Broker) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if cfg, ok := r.configs[slug]; ok {
		return cfg, r.brokers[slug]
	}
	return r.configs[r.flagshipSlug], r.brokers[r.flagshipSlug]
}

// Teams returns the registered slugs in no particular order. Used by the
// provisioning surface and by admin tools.
func (r *Registry) Teams() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.configs))
	for slug := range r.configs {
		out = append(out, slug)
	}
	return out
}

// Stop tears down every team's broker. Used on orderly server shutdown.
func (r *Registry) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.brokers {
		b.Stop()
	}
}
