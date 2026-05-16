package usage

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CodexReader scans OpenAI's `codex` CLI session logs. Default root is
// ~/.codex. Sessions live under <root>/sessions/YYYY/MM/DD/rollout-*.jsonl;
// older versions also wrote flat JSON files in <root>/history/ — we walk
// both subtrees.
//
// Each transcript line is permissively parsed by the shared genericTurn
// helper so OpenAI's `usage.prompt_tokens` / `completion_tokens` shape
// folds into the same Daily bucket the Claude and OpenCode readers produce.
type CodexReader struct {
	Root string
}

func (r CodexReader) Source() Source { return Codex }

func (r CodexReader) Read() ([]Daily, error) {
	root := r.Root
	if root == "" {
		if v := os.Getenv("CODEX_HOME"); v != "" {
			root = v
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, nil
			}
			root = filepath.Join(home, ".codex")
		}
	}
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}

	buckets := map[string]*Daily{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(path, ".jsonl"):
			scanGenericJSONL(path, Codex, buckets)
		case strings.HasSuffix(path, ".json"):
			scanGenericJSONFile(path, Codex, buckets)
		}
		return nil
	})

	out := make([]Daily, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, *b)
	}
	return out, nil
}
