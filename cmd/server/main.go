// vibespace-server — wraps the lobby in an SSH server so anyone can connect
// with `ssh -p 2222 you@host` and land in the chat.
//
// One shared hub.Hub backs every session; each SSH connection gets its own
// app.App (intro + lobby) using the SSH user name as identity.
//
// Configuration via env vars:
//
//	VIBESPACE_ADDR           listen address, default ":2222"
//	VIBESPACE_HOSTKEY        host key path, default ".ssh/id_ed25519" (auto-generated on first run)
//	VIBESPACE_MAX_SESS       max concurrent sessions, default 64
//	VIBESPACE_GH_CLIENT_ID   GitHub OAuth app client id (enables /auth github)
//	VIBESPACE_IDENTITY_PATH  path to identity store JSON, default "./identities.json"
//
// Run on a non-22 port unless you've moved system OpenSSH. Front it with a
// tunnel or VPS proxy before pointing public DNS at your home machine.
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/auth"
	"github.com/bchayka/gitstatus/internal/hub"
	"github.com/bchayka/gitstatus/internal/identity"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

func main() {
	// Under systemd, stdout isn't a TTY so termenv defaults to no-color and
	// lipgloss strips every ANSI escape before it reaches the SSH wire. Force
	// truecolor on the default renderer — modern terminals will downgrade
	// gracefully if the client doesn't support 24-bit color.
	lipgloss.SetColorProfile(termenv.TrueColor)

	addr := envOr("VIBESPACE_ADDR", ":2222")
	hostKey := envOr("VIBESPACE_HOSTKEY", ".ssh/id_ed25519")
	maxSess := envInt("VIBESPACE_MAX_SESS", 64)
	ghClientID := os.Getenv("VIBESPACE_GH_CLIENT_ID")
	identityPath := envOr("VIBESPACE_IDENTITY_PATH", "./identities.json")

	world := hub.New()

	var authSvc *auth.Service
	if ghClientID != "" {
		store, err := identity.Open(identityPath)
		if err != nil {
			log.Fatalf("identity store: %v", err)
		}
		authSvc = auth.New(ghClientID, store)
		log.Printf("github auth enabled, identity store at %s", identityPath)
	} else {
		log.Printf("github auth disabled (set VIBESPACE_GH_CLIENT_ID to enable)")
	}

	var active atomic.Int64

	s, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(hostKey),
		wish.WithIdleTimeout(10*time.Minute),

		// Accept any pubkey — we don't allowlist, we just want to capture the
		// fingerprint for `/auth github` linking. Without a PublicKeyHandler,
		// charm's ssh defaults to NoClientAuth, which means OpenSSH never
		// offers a key in the first place and sess.PublicKey() is always nil.
		wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool {
			return true
		}),
		// Once any auth handler is set, NoClientAuth flips off — so keyless
		// clients also need a path. Keyboard-interactive that returns true
		// without challenging the user gets them through silently. They land
		// as @guest and can't use /auth (no fingerprint to bind to).
		wish.WithKeyboardInteractiveAuth(func(_ ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
			return true
		}),

		wish.WithMiddleware(
			bm.Middleware(teaHandler(world, authSvc, &active, int64(maxSess))),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatalf("wish: %v", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Printf("vibespace-server listening on %s (max %d sessions)", addr, maxSess)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-done
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Printf("shutdown: %v", err)
	}
}

// teaHandler returns a wish bubbletea handler that builds a per-session app.
// Enforces maxSess and pumps lifecycle into App.Cleanup when the session ends.
func teaHandler(world *hub.Hub, authSvc *auth.Service, active *atomic.Int64, maxSess int64) bm.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		_, _, hasPty := sess.Pty()
		if !hasPty {
			wish.Fatalln(sess, "vibespace requires an interactive terminal (try without -T)")
			return nil, nil
		}
		if n := active.Add(1); n > maxSess {
			active.Add(-1)
			wish.Fatalln(sess, "server full — too many concurrent sessions, try again in a moment")
			return nil, nil
		}

		fingerprint := pubkeyFingerprint(sess)
		nick := sshUser(sess)
		// A linked GitHub identity overrides the SSH-derived nick — it's the
		// stronger claim (the user proved control of the GitHub account in a
		// past session).
		if authSvc != nil {
			if gh := authSvc.Lookup(fingerprint); gh != "" {
				nick = "@" + gh
			}
		}

		a := app.New(nick, fingerprint, world, authSvc)

		// Cleanup on session end: SSH closes ctx -> unsubscribe + free slot.
		go func() {
			<-sess.Context().Done()
			a.Cleanup()
			active.Add(-1)
		}()

		return a, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

// pubkeyFingerprint returns the SHA256 fingerprint of the session's public
// key (e.g. "SHA256:abcdef..."), or "" if the client didn't present one.
func pubkeyFingerprint(sess ssh.Session) string {
	pk := sess.PublicKey()
	if pk == nil {
		return ""
	}
	return gossh.FingerprintSHA256(pk)
}

// sshUser derives a display name from the SSH session. Prefers the SSH user
// (`ssh foo@host` -> "@foo"), falling back to a short prefix of the public key
// fingerprint for unauthenticated sessions.
func sshUser(sess ssh.Session) string {
	if u := strings.TrimSpace(sess.User()); u != "" && u != "anonymous" {
		return "@" + sanitizeNick(u)
	}
	if pk := sess.PublicKey(); pk != nil {
		fp := gossh.FingerprintSHA256(pk)
		// fp looks like "SHA256:abcdef..."; trim prefix and shorten.
		if i := strings.IndexByte(fp, ':'); i >= 0 {
			fp = fp[i+1:]
		}
		if len(fp) > 8 {
			fp = fp[:8]
		}
		return "@guest-" + strings.ToLower(fp)
	}
	return "@guest"
}

// sanitizeNick keeps the displayed user predictable: lowercase, alnum/_/-, max
// 24 chars. Don't trust raw SSH usernames in the chat surface.
func sanitizeNick(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
		if b.Len() >= 24 {
			break
		}
	}
	if b.Len() == 0 {
		return "guest"
	}
	return b.String()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
