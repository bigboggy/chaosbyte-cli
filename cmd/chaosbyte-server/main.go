// chaosbyte-server hosts chaosbyte over SSH. Each connection spawns its own
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
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

func main() {
	host := flag.String("host", "0.0.0.0", "SSH listen host")
	port := flag.String("port", "23234", "SSH listen port")
	keyPath := flag.String("hostkey", ".ssh/chaosbyte_ed25519", "SSH host key path (auto-generated if missing)")
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
			bm.Middleware(handlerFor(registry)),
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
	log.Info("starting chaosbyte SSH server", "host", *host, "port", *port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("stopping chaosbyte SSH server")
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
// Per-session: the team's theme is applied just before the program is
// returned so the colors match the team. Two simultaneous SSH sessions
// to different teams therefore call theme.Apply with different palettes;
// this is fine because each session's renderer reads the package-level
// theme vars at View time, and Bubbletea's middleware runs the handler
// once per session before the program starts.
//
// (A future revision will move the palette into per-session render state
// so two teams can run on the same process without theme.Apply races.)
func handlerFor(reg *platform.Registry) bm.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if _, _, active := s.Pty(); !active {
			wish.Fatalln(s, "chaosbyte requires an interactive terminal")
			return nil, nil
		}
		slug := s.User()
		if slug == "" {
			slug = "chaosbyte"
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

