// Package reportcli implements the `vibespace report` subcommand: scan local
// AI-CLI transcripts and pipe a JSON payload over `ssh -T <server> report`.
//
// It lives in its own package so main.go can stay a one-liner dispatcher and
// `go run main.go` works as well as `go run .`.
package reportcli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/bigboggy/vibespace/internal/usage"
)

// defaultServer is where the subcommand uploads when neither --server nor
// $VIBESPACE_SERVER is set. Matches the production instance referenced by
// scripts/deploy.sh.
const defaultServer = "vibespace.sh:2222"

// Run executes the report subcommand. args is the slice of flag args (i.e.
// os.Args[2:] after the "report" verb). Exits the process with a non-zero
// status on failure so callers don't need to inspect errors.
//
// Authentication is delegated to the system ssh client — keys, known_hosts,
// and agent forwarding all behave the way the user already has them
// configured. The server identifies the uploader by SSH pubkey fingerprint
// and refuses entries from keys it hasn't seen do /auth in the TUI.
func Run(args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	server := fs.String("server", "", "vibespace server (host[:port]); falls back to $VIBESPACE_SERVER or "+defaultServer)
	user := fs.String("user", "", "SSH user to connect as; empty uses your SSH config / OS user")
	dryRun := fs.Bool("dry-run", false, "scan transcripts and print the payload JSON to stdout; don't upload")
	keyPath := fs.String("i", "", "SSH identity file to use (passed through to ssh -i)")
	verbose := fs.Bool("v", false, "verbose: print per-source progress")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: vibespace report [flags]")
		fmt.Fprintln(os.Stderr, "Reads ~/.claude, opencode and codex transcripts and uploads daily token totals.")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	addr := strings.TrimSpace(*server)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("VIBESPACE_SERVER"))
	}
	if addr == "" {
		addr = defaultServer
	}

	// Collect entries from every reader. A reader that can't find its CLI
	// returns (nil, nil) — that's normal, not an error.
	var entries []usage.Daily
	for _, r := range usage.AllReaders() {
		ds, err := r.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vibespace report: %s: %v\n", r.Source(), err)
			continue
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "  %-9s %d day(s)\n", r.Source(), len(ds))
		}
		entries = append(entries, ds...)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "vibespace report: nothing to upload — no Claude/OpenCode/Codex transcripts found")
		os.Exit(1)
	}

	// Deterministic order: source then date asc. Makes dry-run diffs sane
	// across invocations and is friendlier for the server log.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].Date < entries[j].Date
	})

	doc := struct {
		Entries []usage.Daily `json:"entries"`
	}{Entries: entries}

	if *dryRun {
		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "vibespace report: encode:", err)
			os.Exit(1)
		}
		os.Stdout.Write(out)
		fmt.Println()
		return
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibespace report: encode:", err)
		os.Exit(1)
	}

	addrUser, host, port, err := splitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibespace report: bad server %q: %v\n", addr, err)
		os.Exit(1)
	}

	sshArgs := []string{"-T"}
	if port != "" {
		sshArgs = append(sshArgs, "-p", port)
	}
	if *keyPath != "" {
		sshArgs = append(sshArgs, "-i", *keyPath)
	}
	// --user takes precedence over a user@ prefix in --server; matches the
	// way `ssh -l` overrides the destination's user portion.
	finalUser := *user
	if finalUser == "" {
		finalUser = addrUser
	}
	target := host
	if finalUser != "" {
		target = finalUser + "@" + host
	}
	sshArgs = append(sshArgs, target, "report")

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if *verbose {
		fmt.Fprintf(os.Stderr, "uploading %d entries to %s ...\n", len(entries), addr)
	}
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		os.Exit(1)
	}
}

// splitHostPort accepts "host", "host:port", "user@host", "user@host:port".
// IPv6 brackets are not supported — we never expect those here. Returns the
// user, host, and port as separate strings (port "" when unspecified).
func splitHostPort(addr string) (user, host, port string, err error) {
	if at := strings.LastIndex(addr, "@"); at >= 0 {
		user = addr[:at]
		addr = addr[at+1:]
	}
	if !strings.Contains(addr, ":") {
		return user, addr, "", nil
	}
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", "", err
	}
	if _, perr := strconv.Atoi(p); perr != nil {
		return "", "", "", fmt.Errorf("port %q not numeric", p)
	}
	return user, h, p, nil
}
