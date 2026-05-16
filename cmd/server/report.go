package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/bigboggy/vibespace/internal/auth"
	"github.com/bigboggy/vibespace/internal/store"
	"github.com/bigboggy/vibespace/internal/usage"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// reportPayloadCap bounds the JSON body the server will read from a report
// session. A year of three sources at one bucket per day is ~3*365 entries
// ~ 100KB worst case; 5 MiB leaves comfortable headroom while denying a
// malicious client from OOMing the process.
const reportPayloadCap = 5 << 20

// reportMiddleware intercepts SSH sessions whose command is `report` and
// upserts their JSON payload of token totals into the store. The middleware
// passes everything else through so the TUI path is unaffected.
//
// Order in main.go is important: this must sit OUTSIDE activeterm (so
// non-PTY report sessions don't get rejected) but INSIDE logging (so the
// access log still records them).
func reportMiddleware(data *store.Store, authSvc *auth.Service) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			cmd := sess.Command()
			if len(cmd) == 0 || cmd[0] != "report" {
				next(sess)
				return
			}
			handleReport(sess, data, authSvc)
		}
	}
}

func handleReport(sess ssh.Session, data *store.Store, authSvc *auth.Service) {
	fp := pubkeyFingerprint(sess)
	if fp == "" {
		fail(sess, "no SSH pubkey presented; can't identify user")
		return
	}
	if authSvc == nil {
		fail(sess, "server has no auth configured; reporting disabled")
		return
	}
	ghLogin := authSvc.Lookup(fp)
	if ghLogin == "" {
		fail(sess, "this SSH key isn't linked yet — open an interactive session and run /auth first")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(sess, reportPayloadCap))
	if err != nil {
		fail(sess, "read payload: "+err.Error())
		return
	}
	if len(payload) == 0 {
		fail(sess, "empty payload — pipe a JSON document on stdin")
		return
	}

	var doc struct {
		Entries []usage.Daily `json:"entries"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		fail(sess, "invalid JSON: "+err.Error())
		return
	}

	var accepted, skipped int
	for _, e := range doc.Entries {
		if err := validateReportEntry(e); err != nil {
			fmt.Fprintf(sess.Stderr(), "skip entry: %v\n", err)
			skipped++
			continue
		}
		if err := data.RecordTokenUsage(store.TokenUsage{
			Login:      ghLogin,
			Source:     store.TokenSource(e.Source),
			Date:       e.Date,
			Input:      e.Input,
			Output:     e.Output,
			CacheWrite: e.CacheWrite,
			CacheRead:  e.CacheRead,
		}); err != nil {
			fail(sess, "db write: "+err.Error())
			return
		}
		accepted++
	}

	log.Printf("report from @%s: accepted=%d skipped=%d", ghLogin, accepted, skipped)
	fmt.Fprintf(sess, "ok: @%s · accepted %d · skipped %d\n", ghLogin, accepted, skipped)
	_ = sess.Exit(0)
}

func fail(sess ssh.Session, msg string) {
	fmt.Fprintln(sess.Stderr(), "vibespace report:", msg)
	_ = sess.Exit(1)
}

func validateReportEntry(e usage.Daily) error {
	if e.Date == "" {
		return errors.New("missing date")
	}
	if _, err := time.Parse("2006-01-02", e.Date); err != nil {
		return fmt.Errorf("date %q not YYYY-MM-DD", e.Date)
	}
	switch e.Source {
	case usage.Claude, usage.OpenCode, usage.Codex:
	default:
		return fmt.Errorf("unknown source %q", e.Source)
	}
	if e.Input < 0 || e.Output < 0 || e.CacheWrite < 0 || e.CacheRead < 0 {
		return errors.New("negative token count")
	}
	if e.Total() == 0 {
		return errors.New("all-zero entry")
	}
	return nil
}
