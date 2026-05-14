package lobby

import (
	"context"
	"fmt"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// authFlowState holds the per-session in-progress device flow. nil means no
// flow is active; the lobby renders the modal whenever this is non-nil.
//
// ctx/cancel cover the whole flow (Start + Poll). Esc and Cleanup both call
// cancel, which propagates into the HTTP request currently in flight on
// GitHub's poll endpoint and unblocks it with ctx.Canceled.
type authFlowState struct {
	userCode  string
	verifyURL string
	status    string // human-readable line under the code
	failed    bool   // true once a terminal error landed; esc dismisses
	ctx       context.Context
	cancel    context.CancelFunc
}

// authStartedMsg lands after the initial /login/device/code call succeeds.
type authStartedMsg struct {
	deviceCode string
	userCode   string
	verifyURL  string
	interval   time.Duration
}

// authResultMsg lands when polling completes (success or terminal error). On
// success ghUser is the resolved GitHub login.
type authResultMsg struct {
	ghUser string
	err    error
}

// cmdAuthGithub runs /auth github. Returns a Cmd that fires the device-code
// request; subsequent steps are driven by Update on authStartedMsg.
func (s *Screen) cmdAuthGithub() (*Screen, tea.Cmd) {
	if s.auth == nil {
		s.postSystem("github auth isn't configured on this server")
		return s, nil
	}
	if s.ghLogin != "" {
		s.postSystem(fmt.Sprintf("already authenticated as @%s — /logout to unlink", s.ghLogin))
		return s, nil
	}
	if s.fingerprint == "" {
		s.postSystem("you connected without an SSH key — reconnect with `ssh -i <key>` to use /auth")
		return s, nil
	}
	if s.authFlow != nil {
		s.postSystem("auth already in progress — press esc to cancel")
		return s, nil
	}

	// Reserve a cancelable context for the whole flow. Stored on authFlow so
	// esc or Cleanup() can abort both Start and Poll.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	s.authFlow = &authFlowState{
		status: "contacting github...",
		ctx:    ctx,
		cancel: cancel,
	}

	svc := s.auth
	return s, func() tea.Msg {
		r, err := svc.StartFlow(ctx)
		if err != nil {
			return authResultMsg{err: err}
		}
		return authStartedMsg{
			deviceCode: r.DeviceCode,
			userCode:   r.UserCode,
			verifyURL:  r.VerificationURI,
			interval:   r.Interval,
		}
	}
}

// pollAuthCmd waits for the user to authorize on github.com. The ctx comes
// from authFlow so esc/Cleanup cancel the in-flight HTTP request.
func (s *Screen) pollAuthCmd(ctx context.Context, deviceCode string, interval time.Duration) tea.Cmd {
	svc := s.auth
	return func() tea.Msg {
		ghUser, err := svc.PollFlow(ctx, deviceCode, interval)
		return authResultMsg{ghUser: ghUser, err: err}
	}
}

// handleAuthStarted transitions from "contacting github" to the live modal
// with the user code, and fires the poll cmd.
func (s *Screen) handleAuthStarted(m authStartedMsg) (*Screen, tea.Cmd) {
	if s.authFlow == nil {
		// Cancelled before the call returned; nothing to do.
		return s, nil
	}
	s.authFlow.userCode = m.userCode
	s.authFlow.verifyURL = m.verifyURL
	s.authFlow.status = "waiting for you to authorize on github..."
	return s, s.pollAuthCmd(s.authFlow.ctx, m.deviceCode, m.interval)
}

// handleAuthResult applies the terminal state of an auth flow. On success it
// mutates meUser, persists the mapping, and announces the link in chat. On
// failure it leaves the modal up with the error so the user can read it; esc
// dismisses.
func (s *Screen) handleAuthResult(m authResultMsg) (*Screen, tea.Cmd) {
	if s.authFlow == nil {
		// User cancelled; ignore late delivery.
		return s, nil
	}
	if m.err != nil {
		s.authFlow.status = "failed: " + m.err.Error()
		s.authFlow.failed = true
		return s, nil
	}

	prev := s.meUser
	newNick := "@" + m.ghUser

	if err := s.auth.Link(s.fingerprint, m.ghUser); err != nil {
		s.authFlow.status = "failed to persist: " + err.Error()
		s.authFlow.failed = true
		return s, nil
	}

	s.meUser = newNick
	s.ghLogin = m.ghUser

	if s.authFlow.cancel != nil {
		s.authFlow.cancel()
	}
	s.authFlow = nil

	s.hub.Post(s.activeName, "*",
		fmt.Sprintf("%s authenticated as %s via github", prev, newNick),
		ui.ChatSystem)
	return s, nil
}

// cancelAuthFlow aborts an in-progress flow. Safe to call when authFlow is nil.
func (s *Screen) cancelAuthFlow() {
	if s.authFlow == nil {
		return
	}
	if s.authFlow.cancel != nil {
		s.authFlow.cancel()
	}
	s.authFlow = nil
}

// renderAuthModal draws the centered card shown during a device flow.
func (s *Screen) renderAuthModal(width, height int) string {
	a := s.authFlow
	st := s.styles

	titleColor := st.Accent2
	if a.failed {
		titleColor = st.Warn
	}
	title := st.NewStyle().Foreground(titleColor).Bold(true).
		Render("Link your GitHub account")

	var body []string
	if a.userCode == "" {
		body = append(body, st.NewStyle().Foreground(st.Muted).Render(a.status))
	} else {
		body = append(body,
			"1. Open "+st.NewStyle().Foreground(st.Accent).Bold(true).Render(a.verifyURL),
			"2. Enter code: "+st.NewStyle().Foreground(st.OK).Bold(true).Render(a.userCode),
			"",
			st.NewStyle().Foreground(st.Muted).Italic(true).Render(a.status),
		)
	}

	hint := "press esc to cancel"
	if a.failed {
		hint = "press esc to dismiss"
	}
	body = append(body, "", st.NewStyle().Foreground(st.Muted).Render(hint))

	inner := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, body...)...)
	card := st.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(titleColor).
		Padding(1, 3).
		Render(inner)
	return st.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}
