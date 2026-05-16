package usage

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// OpenCodeReader walks OpenCode's session storage. The on-disk layout has
// drifted across OpenCode versions; this reader looks at the two stable
// shapes:
//
//  1. Per-message JSON files under <root>/storage/message/<session>/<id>.json
//     where each file carries a `tokens` or `usage` block on assistant turns.
//  2. Per-session JSONL files under <root>/storage/session/ where each line
//     is a turn.
//
// Default root resolution: $OPENCODE_DATA → $XDG_DATA_HOME/opencode →
// platform default (~/.local/share/opencode or
// ~/Library/Application Support/opencode).
type OpenCodeReader struct {
	Root string
}

func (r OpenCodeReader) Source() Source { return OpenCode }

func (r OpenCodeReader) Read() ([]Daily, error) {
	root := r.Root
	if root == "" {
		root = defaultOpenCodeRoot()
	}
	if root == "" {
		return nil, nil
	}
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}

	buckets := map[string]*Daily{}
	// Look at both the message/ and session/ subtrees — different versions
	// stash data in one or the other.
	for _, sub := range []string{"storage/message", "storage/session", "storage"} {
		dir := filepath.Join(root, sub)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			switch {
			case strings.HasSuffix(path, ".jsonl"):
				scanGenericJSONL(path, OpenCode, buckets)
			case strings.HasSuffix(path, ".json"):
				scanGenericJSONFile(path, OpenCode, buckets)
			}
			return nil
		})
	}

	out := make([]Daily, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, *b)
	}
	return out, nil
}

func defaultOpenCodeRoot() string {
	if v := os.Getenv("OPENCODE_DATA"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "opencode")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "opencode")
	default:
		return filepath.Join(home, ".local", "share", "opencode")
	}
}

// ── generic JSON helpers shared by opencode + codex readers ────────────────

// genericTurn is intentionally permissive — it accepts the union of the
// Anthropic-style `usage` block (input/output_tokens), the OpenAI-style
// `usage` block (prompt/completion_tokens), and OpenCode's `tokens` block.
// Fields not present unmarshal as zero and are skipped.
type genericTurn struct {
	Type      string  `json:"type,omitempty"`
	Role      string  `json:"role,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	Time      *int64  `json:"time,omitempty"` // unix-ms (opencode)
	Date      string  `json:"date,omitempty"`

	Usage *struct {
		InputTokens             int64 `json:"input_tokens"`
		OutputTokens            int64 `json:"output_tokens"`
		PromptTokens            int64 `json:"prompt_tokens"`
		CompletionTokens        int64 `json:"completion_tokens"`
		CacheCreationInputToks  int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens    int64 `json:"cache_read_input_tokens"`
	} `json:"usage,omitempty"`

	Tokens *struct {
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     *struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens,omitempty"`

	// Some session files wrap the assistant turn inside a `message` object,
	// matching the Claude transcript layout.
	Message *struct {
		Role  string `json:"role"`
		Usage *struct {
			InputTokens             int64 `json:"input_tokens"`
			OutputTokens            int64 `json:"output_tokens"`
			CacheCreationInputToks  int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens    int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
}

// applyGenericTurn folds one parsed turn into the per-day buckets. Returns
// false if the turn has no usable usage data so the caller can fall back to
// a different shape.
func applyGenericTurn(t genericTurn, src Source, date string, buckets map[string]*Daily) bool {
	var in, out, cw, cr int64
	switch {
	case t.Usage != nil:
		// Anthropic-style first; OpenAI-style as fallback.
		in = t.Usage.InputTokens + t.Usage.PromptTokens
		out = t.Usage.OutputTokens + t.Usage.CompletionTokens
		cw = t.Usage.CacheCreationInputToks
		cr = t.Usage.CacheReadInputTokens
	case t.Message != nil && t.Message.Usage != nil:
		in = t.Message.Usage.InputTokens
		out = t.Message.Usage.OutputTokens
		cw = t.Message.Usage.CacheCreationInputToks
		cr = t.Message.Usage.CacheReadInputTokens
	case t.Tokens != nil:
		in = t.Tokens.Input
		out = t.Tokens.Output + t.Tokens.Reasoning
		if t.Tokens.Cache != nil {
			cw = t.Tokens.Cache.Write
			cr = t.Tokens.Cache.Read
		}
	default:
		return false
	}
	if in+out+cw+cr == 0 {
		return false
	}
	b, ok := buckets[date]
	if !ok {
		b = &Daily{Source: src, Date: date}
		buckets[date] = b
	}
	b.Input += in
	b.Output += out
	b.CacheWrite += cw
	b.CacheRead += cr
	return true
}

// turnDate picks the best date for a turn out of the available fields. Falls
// back to fallbackDate (typically the file's modtime) when nothing is set.
func turnDate(t genericTurn, fallbackDate string) string {
	if d := utcDate(t.Timestamp); d != "" {
		return d
	}
	if d := utcDate(t.CreatedAt); d != "" {
		return d
	}
	if t.Time != nil && *t.Time > 0 {
		ms := *t.Time
		return time.UnixMilli(ms).UTC().Format("2006-01-02")
	}
	if t.Date != "" {
		return t.Date
	}
	return fallbackDate
}

func scanGenericJSONL(path string, src Source, buckets map[string]*Daily) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fallback := fileMtimeDate(f)

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<16), 16<<20)
	for sc.Scan() {
		var t genericTurn
		if err := json.Unmarshal(sc.Bytes(), &t); err != nil {
			continue
		}
		date := turnDate(t, fallback)
		if date == "" {
			continue
		}
		applyGenericTurn(t, src, date, buckets)
	}
}

func scanGenericJSONFile(path string, src Source, buckets map[string]*Daily) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Try single-object first.
	var t genericTurn
	if err := json.Unmarshal(data, &t); err == nil {
		date := turnDate(t, fileMtimeDateFromPath(path))
		if date != "" && applyGenericTurn(t, src, date, buckets) {
			return
		}
	}
	// Some opencode versions store an array of messages per file.
	var arr []genericTurn
	if err := json.Unmarshal(data, &arr); err != nil {
		return
	}
	fallback := fileMtimeDateFromPath(path)
	for _, t := range arr {
		date := turnDate(t, fallback)
		if date == "" {
			continue
		}
		applyGenericTurn(t, src, date, buckets)
	}
}

func fileMtimeDate(f *os.File) string {
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	return st.ModTime().UTC().Format("2006-01-02")
}

func fileMtimeDateFromPath(path string) string {
	st, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return st.ModTime().UTC().Format("2006-01-02")
}
