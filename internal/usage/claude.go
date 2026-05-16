package usage

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeReader scans Claude Code's transcript directory. Layout (stable
// since CC's 2024 launch):
//
//	~/.claude/projects/<sanitized-project-path>/<session-uuid>.jsonl
//
// Each line is a JSON-encoded turn. We only care about assistant turns,
// which carry the model's reported token usage on their `message.usage`
// field (Anthropic API shape).
type ClaudeReader struct {
	// Root overrides the default ~/.claude lookup; empty means auto-detect.
	Root string
}

func (r ClaudeReader) Source() Source { return Claude }

func (r ClaudeReader) Read() ([]Daily, error) {
	root := r.Root
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil
		}
		root = filepath.Join(home, ".claude")
	}
	projectsDir := filepath.Join(root, "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		// Not installed (or no transcripts yet). Silent skip.
		return nil, nil
	}

	buckets := map[string]*Daily{}
	walkErr := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Don't abort the whole scan on one bad path — just skip.
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		scanClaudeFile(path, buckets)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	out := make([]Daily, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, *b)
	}
	return out, nil
}

// claudeTurn is the subset of a Claude Code transcript line we care about.
// Other fields (content blocks, tool calls, parent_uuid, ...) are ignored.
type claudeTurn struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   *struct {
		Usage *struct {
			Input                     int64 `json:"input_tokens"`
			Output                    int64 `json:"output_tokens"`
			CacheCreationInputTokens  int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens      int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func scanClaudeFile(path string, buckets map[string]*Daily) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// Transcript lines can be large when assistant turns include long
	// content blocks; bump the buffer ceiling so we don't silently truncate.
	sc.Buffer(make([]byte, 0, 1<<16), 16<<20)
	for sc.Scan() {
		var t claudeTurn
		if err := json.Unmarshal(sc.Bytes(), &t); err != nil {
			continue
		}
		if t.Type != "assistant" || t.Message == nil || t.Message.Usage == nil {
			continue
		}
		date := utcDate(t.Timestamp)
		if date == "" {
			continue
		}
		b, ok := buckets[date]
		if !ok {
			b = &Daily{Source: Claude, Date: date}
			buckets[date] = b
		}
		u := t.Message.Usage
		b.Input += u.Input
		b.Output += u.Output
		b.CacheWrite += u.CacheCreationInputTokens
		b.CacheRead += u.CacheReadInputTokens
	}
}

// utcDate parses an RFC3339-ish timestamp and returns YYYY-MM-DD in UTC.
// Returns "" when the timestamp doesn't parse — the caller skips that turn.
func utcDate(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		// Some transcripts use a shorter precision; try the basic form.
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ""
		}
	}
	return t.UTC().Format("2006-01-02")
}
