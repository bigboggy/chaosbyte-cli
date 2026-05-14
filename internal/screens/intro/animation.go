package intro

import (
	"math"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// Phase boundaries in milliseconds since intro start. The "hold" window is
// deliberately the longest phase — the logo is the brand moment.
const (
	phaseBootEnd   = 400  // boot lines type out
	phaseBuildEnd  = 1000 // logo builds line by line (600ms)
	phaseHoldEnd   = 3500 // logo holds with tagline (2500ms — the brand beat)
	phaseShrinkEnd = 3800 // collapses to plain "VIBESPACE"
	phaseByteEnd   = 4200 // morphs through binary → "space"
	phaseBlockEnd  = 4400 // single block
	phaseFadeEnd   = 4600 // blank, then transition
)

var bootLines = []string{
	"vibespace boot v0.1.0",
	"",
	"[ok] kernel              loaded",
	"[ok] mesh.vibespace.dev  online",
	"[ok] vibes               synced",
	"[ok] tui driver          initialized",
	"[ok] #lobby              ready",
}

// View renders the current animation frame, centered in the given viewport.
func (s *Screen) View(width, height int) string {
	ms := int(time.Since(s.start).Milliseconds())

	var content string
	switch {
	case ms < phaseBootEnd:
		content = renderBoot(s.styles, ms)
	case ms < phaseBuildEnd:
		content = renderBuild(s.styles, ms-phaseBootEnd)
	case ms < phaseHoldEnd:
		content = renderHold(s.styles, ms-phaseBuildEnd)
	case ms < phaseShrinkEnd:
		content = renderShrink(s.styles, ms-phaseHoldEnd)
	case ms < phaseByteEnd:
		content = renderByte(s.styles, ms-phaseShrinkEnd)
	case ms < phaseBlockEnd:
		content = renderBlock(s.styles, ms-phaseByteEnd)
	default:
		content = ""
	}

	skip := s.styles.NewStyle().Foreground(s.styles.Muted).Italic(true).
		Render("press any key to skip")
	frame := lipgloss.JoinVertical(lipgloss.Center, content, "", "", skip)
	return s.styles.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
}

func renderBoot(st *theme.Styles, ms int) string {
	revealed := ms / 55
	if revealed > len(bootLines) {
		revealed = len(bootLines)
	}
	var out []string
	for i := 0; i < revealed; i++ {
		line := bootLines[i]
		style := st.NewStyle().Foreground(st.OK)
		if strings.HasPrefix(line, "vibespace") {
			style = st.NewStyle().Foreground(st.Accent2).Bold(true)
		}
		out = append(out, style.Render(line))
	}
	if revealed < len(bootLines) && ms%500 < 250 {
		out = append(out, st.NewStyle().Foreground(st.Accent).Render("█"))
	}
	return strings.Join(out, "\n")
}

func renderBuild(st *theme.Styles, ms int) string {
	revealed := ms/100 + 1
	if revealed > len(theme.LogoLines) {
		revealed = len(theme.LogoLines)
	}
	gradient := st.LogoGradient()
	var out []string
	for i := 0; i < revealed; i++ {
		out = append(out, st.NewStyle().
			Foreground(gradient[i%len(gradient)]).
			Bold(true).
			Render(theme.LogoLines[i]))
	}
	return strings.Join(out, "\n")
}

func renderHold(st *theme.Styles, ms int) string {
	logo := st.RenderLogo()
	pulse := math.Abs(math.Sin(float64(ms) / 180.0))
	color := st.Accent
	if pulse > 0.5 {
		color = st.Accent2
	}
	tagline := st.NewStyle().Foreground(color).Italic(true).
		Render("an all-in-one place for devs and vibe coders")
	dots := st.NewStyle().Foreground(st.Muted).
		Render(strings.Repeat(".", (ms/200)%4))
	connecting := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render("connecting to #lobby") + dots
	return lipgloss.JoinVertical(lipgloss.Center, logo, "", tagline, "", connecting)
}

func renderShrink(st *theme.Styles, ms int) string {
	progress := float64(ms) / float64(phaseShrinkEnd-phaseHoldEnd)
	if progress < 0.4 {
		mid := []string{
			theme.LogoLines[1], theme.LogoLines[2], theme.LogoLines[3], theme.LogoLines[4],
		}
		return st.NewStyle().Foreground(st.Accent2).Bold(true).
			Render(strings.Join(mid, "\n"))
	}
	if progress < 0.7 {
		mid := []string{theme.LogoLines[2], theme.LogoLines[3]}
		return st.NewStyle().Foreground(st.Accent2).Bold(true).
			Render(strings.Join(mid, "\n"))
	}
	return st.NewStyle().Foreground(st.Accent2).Bold(true).
		Render("V I B E S P A C E")
}

func renderByte(st *theme.Styles, ms int) string {
	progress := float64(ms) / float64(phaseByteEnd-phaseShrinkEnd)
	if progress < 0.3 {
		return st.NewStyle().Foreground(st.Accent).Bold(true).
			Render("00100000")
	}
	if progress < 0.7 {
		return st.NewStyle().Foreground(st.Accent).Bold(true).
			Render("space")
	}
	return st.NewStyle().Foreground(st.Accent).Bold(true).
		Render("s")
}

func renderBlock(st *theme.Styles, ms int) string {
	if ms%140 < 70 {
		return st.NewStyle().Foreground(st.Accent).Render("▪")
	}
	return st.NewStyle().Foreground(st.Muted).Render("·")
}
