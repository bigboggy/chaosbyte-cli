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
	"github.com/bchayka/gitstatus/internal/room"
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
	flag.Parse()

	broker := room.New()
	defer broker.Stop()

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(*host, *port)),
		wish.WithHostKeyPath(*keyPath),
		wish.WithMiddleware(
			bm.Middleware(handlerFor(broker)),
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

func handlerFor(broker *room.Broker) bm.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if _, _, active := s.Pty(); !active {
			wish.Fatalln(s, "chaosbyte requires an interactive terminal")
			return nil, nil
		}
		nick := s.User()
		if nick == "" {
			nick = "anonymous"
		}
		return app.New("@"+nick, broker), []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		}
	}
}
