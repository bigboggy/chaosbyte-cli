// vibespace-server hosts vibespace over SSH. Each connection spawns its own
// bubbletea program backed by an app.App; today the rooms are independent
// per session, broker-backed shared state lands as a follow-up commit.
package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/platform"
	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

func main() {
	host := flag.String("host", "0.0.0.0", "SSH listen host")
	port := flag.String("port", "23234", "SSH listen port")
	keyPath := flag.String("hostkey", ".ssh/vibespace_ed25519", "SSH host key path (auto-generated if missing)")
	configsDir := flag.String("configs", "configs", "directory containing per-team .toml config files")
	flag.Parse()

	registry := platform.NewRegistry()
	if loaded, err := config.LoadFromDir(*configsDir); err != nil {
		log.Warn("could not read configs directory", "dir", *configsDir, "error", err)
	} else {
		for _, cfg := range loaded {
			registry.Register(cfg)
			log.Info("registered team", "slug", cfg.Slug, "brand", cfg.Brand.Name)
		}
	}
	defer registry.Stop()

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(*host, *port)),
		wish.WithHostKeyPath(*keyPath),
		wish.WithMiddleware(
			// TrueColor floor: bm.MakeRenderer downgrades to the session
			// context's minColorProfile, which defaults to Ascii. Raising
			// the floor keeps the team palette intact for any modern
			// terminal client.
			bm.MiddlewareWithColorProfile(handlerFor(registry), termenv.TrueColor),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("starting vibespace SSH server", "host", *host, "port", *port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("stopping vibespace SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

// handlerFor returns the Wish bubbletea handler that routes every incoming
// SSH session to the team the user is asking for. The SSH user (the part
// before @ in `ssh user@host`) is the team slug. Unknown slugs land on
// the flagship.
//
// Two pieces of session-scoped state are set up before the program runs:
//
//  1. lipgloss's default renderer is rebound to the SSH session's PTY via
//     bm.MakeRenderer. Without this, the renderer inherits the daemon's
//     non-TTY stdout, detects Ascii, and silently strips every color the
//     theme tries to apply.
//  2. theme.Apply replaces the package-level palette with the team's
//     colors so screens read the right values during View().
//
// Both are process-global and racy across concurrent sessions to different
// teams. The current platform invariant is one server, one team, which
// keeps this safe; per-session render state is the follow-up when the
// platform fans out to multi-team co-tenancy.
func handlerFor(reg *platform.Registry) bm.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if _, _, active := s.Pty(); !active {
			wish.Fatalln(s, "vibespace requires an interactive terminal")
			return nil, nil
		}
		lipgloss.SetDefaultRenderer(bm.MakeRenderer(s))
		slug := s.User()
		if slug == "" {
			slug = "vibespace"
		}
		cfg, broker := reg.Resolve(slug)
		theme.Apply(theme.Palette{
			Bg:       cfg.Theme.Bg,
			Fg:       cfg.Theme.Fg,
			Muted:    cfg.Theme.Muted,
			Accent:   cfg.Theme.Accent,
			Accent2:  cfg.Theme.Accent2,
			BorderHi: cfg.Theme.BorderHi,
			BorderLo: cfg.Theme.BorderLo,
		})
		nick := "@" + slug
		return app.New(nick, broker, cfg), []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		}
	}
}

