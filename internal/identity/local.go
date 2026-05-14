package identity

import "github.com/google/uuid"

// LocalPrincipal returns the sentinel Principal used by the local
// single-user binary (cmd/vibespace). No SSH, no real auth: the local
// user is always "@boggy" with full access to the flagship team.
//
// This is intentionally not gated behind a build tag or env var. The
// local binary is for development; production deployments use
// cmd/vibespace-server with real pubkey auth.
func LocalPrincipal() Principal {
	return Principal{
		ID:          "local:boggy",
		DisplayName: "@boggy",
		Teams:       []string{"vibespace", "monobyte"},
		Roles:       []string{"operator"},
		SessionID:   uuid.New(),
		Kind:        KindHuman,
	}
}

// AllowlistEntryFor exposes the public fields of an allowlist entry to
// callers in cmd/vibespace-server that need to build a Principal from
// a pubkey-verified session. The package-private allowlistEntry stays
// internal; this accessor returns the read-only view.
func (a *Allowlist) PrincipalFor(entry allowlistEntry, sessionID uuid.UUID) Principal {
	return Principal{
		ID:          IDFromPubkey(entry.PubKey),
		DisplayName: entry.Display,
		PublicKey:   entry.PubKey,
		Teams:       entry.Teams,
		Roles:       entry.Roles,
		SessionID:   sessionID,
		Kind:        KindHuman,
	}
}
