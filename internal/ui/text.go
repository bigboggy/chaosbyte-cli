package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Wrap word-wraps s to width columns, preserving paragraph breaks. width<4 is
// clamped to 4 to avoid pathological output.
func Wrap(s string, width int) string {
	if width < 4 {
		width = 4
	}
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		var line string
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if lipgloss.Width(line)+1+lipgloss.Width(w) > width {
				out = append(out, line)
				line = w
			} else {
				line += " " + w
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// Truncate shortens s to at most n cells, ending with an ellipsis if cut.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// HumanizeTime renders a duration since t as a compact relative string.
func HumanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// FakeSHA returns a 7-character pseudo-SHA derived from the current nanosecond
// timestamp. Stable enough that two near-simultaneous calls don't collide.
func FakeSHA() string {
	const hex = "0123456789abcdef"
	now := time.Now().UnixNano()
	out := make([]byte, 7)
	for i := range out {
		out[i] = hex[now&0xf]
		now >>= 4
	}
	return string(out)
}
