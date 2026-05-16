// vibespace — a TUI lobby for devs and vibe coders.
//
// This is the local-mode entrypoint: one user, one hub, one bubbletea program.
// The SSH-server entrypoint lives in cmd/server.
package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/bigboggy/vibespace/internal/app"
	"github.com/bigboggy/vibespace/internal/auth"
	"github.com/bigboggy/vibespace/internal/hub"
	"github.com/bigboggy/vibespace/internal/identity"
	"github.com/bigboggy/vibespace/internal/store"
	"github.com/bigboggy/vibespace/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	h := hub.New()
	// Local-mode SQLite lives under the user's config dir so profiles persist
	// across runs without polluting the working directory.
	dbPath := localDBPath()
	data, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: store: %v\n", err)
		os.Exit(1)
	}
	defer data.Close()

	// Optional /auth in local mode — set VIBESPACE_GH_CLIENT_ID to enable.
	// The identity store sits next to the SQLite DB. With no client id, /auth
	// stays disabled and the lobby is just a local chat surface.
	authSvc, ghLogin := localAuth(data)

	// In local mode the "fingerprint" is the synthesized identity key (see
	// localFingerprint) — only meaningful when authSvc is non-nil. Pass it
	// either way; the lobby just stashes it.
	fingerprint := ""
	if authSvc != nil {
		fingerprint = localFingerprint()
	}

	styles := theme.New(lipgloss.DefaultRenderer(), theme.Default())
	p := tea.NewProgram(
		app.New(styles, localUser(), fingerprint, ghLogin, h, authSvc, data),
		tea.WithAltScreen(), tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: %v\n", err)
		os.Exit(1)
	}
}

func localDBPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		root := dir + "/vibespace"
		_ = os.MkdirAll(root, 0o700)
		return root + "/vibespace.db"
	}
	return "./vibespace.db"
}

// localAuth wires the /auth flow in local mode when VIBESPACE_GH_CLIENT_ID is
// set. With no SSH fingerprint to key off of, the identity store is keyed by
// the OS username instead — so re-running the binary picks up the same link.
func localAuth(data *store.Store) (*auth.Service, string) {
	clientID := os.Getenv("VIBESPACE_GH_CLIENT_ID")
	if clientID == "" {
		return nil, ""
	}
	idPath := localIdentityPath()
	idStore, err := identity.Open(idPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: identity: %v\n", err)
		return nil, ""
	}
	svc := auth.New(clientID, idStore, data)
	if svc == nil {
		return nil, ""
	}
	// Pre-resolve any prior link so the session starts already authenticated.
	ghLogin, _ := idStore.Lookup(localFingerprint())
	return svc, ghLogin
}

func localIdentityPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir + "/vibespace/identities.json"
	}
	return "./identities.json"
}

// localFingerprint is the stable key local-mode sessions use in the identity
// store. The real server uses SHA256 SSH pubkey fingerprints; locally we have
// no SSH layer, so we synthesize one from the OS user. The lobby passes this
// through to auth.Service.Link / .Lookup.
func localFingerprint() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "local:" + u.Username
	}
	return "local:unknown"
}

func localUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "@" + u.Username
	}
	return "@local"
}
