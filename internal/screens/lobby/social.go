package lobby

import (
	"fmt"
	"strings"

	"github.com/bigboggy/vibespace/internal/screens"
	"github.com/bigboggy/vibespace/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

// normLogin trims a leading "@" and lowercases. Both forms work in commands.
func normLogin(s string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(s), "@"))
}

// requireAuth is a precondition for any command that mutates a user's social
// graph. Returns "" + posts a system hint when the session isn't linked yet.
func (s *Screen) requireAuth(action string) string {
	if s.ghLogin == "" {
		s.postSystem("type /auth to link your GitHub account first — " + action + " needs an identity")
		return ""
	}
	return s.ghLogin
}

// requireStore reports whether the data store is configured. Local mode wires
// one in, server mode wires one in, but be defensive in case a future call
// site forgets.
func (s *Screen) requireStore(action string) bool {
	if s.data == nil {
		s.postSystem(action + " isn't available in this build")
		return false
	}
	return true
}

// cmdProfile opens the profile screen. No arg → own profile. Anyone can view;
// the screen itself nudges unauthenticated viewers toward /auth.
func (s *Screen) cmdProfile(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/profile") {
		return s, nil
	}
	target := ""
	switch {
	case len(args) >= 1:
		target = normLogin(args[0])
	case s.ghLogin != "":
		target = s.ghLogin
	default:
		s.postSystem("usage: /profile @user  (or /auth first to view your own)")
		return s, nil
	}
	if target == "" {
		s.postSystem("usage: /profile @user")
		return s, nil
	}
	return s, screens.OpenProfile(target, s.ghLogin)
}

func (s *Screen) cmdFriend(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/friend") {
		return s, nil
	}
	me := s.requireAuth("/friend")
	if me == "" {
		return s, nil
	}
	if len(args) == 0 {
		s.postSystem("usage: /friend @user")
		return s, nil
	}
	target := normLogin(args[0])
	if err := s.data.SendFriendRequest(me, target); err != nil {
		s.postSystem("/friend: " + err.Error())
		return s, nil
	}
	s.postSystem(fmt.Sprintf("friend request sent to @%s", target))
	return s, nil
}

func (s *Screen) cmdAccept(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/accept") {
		return s, nil
	}
	me := s.requireAuth("/accept")
	if me == "" {
		return s, nil
	}
	if len(args) == 0 {
		s.postSystem("usage: /accept @user")
		return s, nil
	}
	from := normLogin(args[0])
	if err := s.data.AcceptFriendRequest(me, from); err != nil {
		s.postSystem("/accept: " + err.Error())
		return s, nil
	}
	s.postSystem(fmt.Sprintf("you and @%s are now friends", from))
	return s, nil
}

func (s *Screen) cmdReject(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/reject") {
		return s, nil
	}
	me := s.requireAuth("/reject")
	if me == "" {
		return s, nil
	}
	if len(args) == 0 {
		s.postSystem("usage: /reject @user")
		return s, nil
	}
	from := normLogin(args[0])
	// Only drop if it's actually a pending-in row — don't let /reject silently
	// unfriend someone.
	st, _ := s.data.FriendStatusBetween(me, from)
	if st != store.FriendPendingIn {
		s.postSystem(fmt.Sprintf("no pending request from @%s", from))
		return s, nil
	}
	if err := s.data.RemoveFriendship(me, from); err != nil {
		s.postSystem("/reject: " + err.Error())
		return s, nil
	}
	s.postSystem(fmt.Sprintf("rejected @%s's friend request", from))
	return s, nil
}

func (s *Screen) cmdUnfriend(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/unfriend") {
		return s, nil
	}
	me := s.requireAuth("/unfriend")
	if me == "" {
		return s, nil
	}
	if len(args) == 0 {
		s.postSystem("usage: /unfriend @user")
		return s, nil
	}
	target := normLogin(args[0])
	st, _ := s.data.FriendStatusBetween(me, target)
	if st != store.FriendAccepted && st != store.FriendPendingOut {
		s.postSystem(fmt.Sprintf("you're not connected to @%s", target))
		return s, nil
	}
	if err := s.data.RemoveFriendship(me, target); err != nil {
		s.postSystem("/unfriend: " + err.Error())
		return s, nil
	}
	s.postSystem(fmt.Sprintf("removed @%s from your friends", target))
	return s, nil
}

func (s *Screen) cmdFriends() (*Screen, tea.Cmd) {
	if !s.requireStore("/friends") {
		return s, nil
	}
	me := s.requireAuth("/friends")
	if me == "" {
		return s, nil
	}
	friends, _ := s.data.Friends(me)
	incoming, _ := s.data.IncomingRequests(me)
	outgoing, _ := s.data.OutgoingRequests(me)

	var lines []string
	if len(incoming) > 0 {
		lines = append(lines, fmt.Sprintf("incoming requests (%d):", len(incoming)))
		for _, f := range incoming {
			lines = append(lines, "  @"+f.Login+"  (/accept @"+f.Login+" or /reject @"+f.Login+")")
		}
	}
	if len(outgoing) > 0 {
		lines = append(lines, fmt.Sprintf("outgoing requests (%d):", len(outgoing)))
		for _, f := range outgoing {
			lines = append(lines, "  @"+f.Login+"  (pending)")
		}
	}
	if len(friends) == 0 && len(incoming) == 0 && len(outgoing) == 0 {
		s.postSystem("no friends yet — /friend @user to send a request")
		return s, nil
	}
	lines = append(lines, fmt.Sprintf("friends (%d):", len(friends)))
	if len(friends) == 0 {
		lines = append(lines, "  (none yet)")
	}
	for _, f := range friends {
		lines = append(lines, "  @"+f.Login)
	}
	s.postSystem(strings.Join(lines, "\n"))
	return s, nil
}

func (s *Screen) cmdPost(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/post") {
		return s, nil
	}
	me := s.requireAuth("/post")
	if me == "" {
		return s, nil
	}
	body := strings.TrimSpace(strings.Join(args, " "))
	if body == "" {
		s.postSystem("usage: /post <message>")
		return s, nil
	}
	if _, err := s.data.CreatePost(me, body); err != nil {
		s.postSystem("/post: " + err.Error())
		return s, nil
	}
	s.postSystem("posted to your profile — /profile to see it")
	return s, nil
}

// cmdSign is the only writer that gates on friendship — guestbook entries
// can only come from accepted friends. /sign @user <message>.
func (s *Screen) cmdSign(args []string) (*Screen, tea.Cmd) {
	if !s.requireStore("/sign") {
		return s, nil
	}
	me := s.requireAuth("/sign")
	if me == "" {
		return s, nil
	}
	if len(args) < 2 {
		s.postSystem("usage: /sign @user <message>")
		return s, nil
	}
	owner := normLogin(args[0])
	body := strings.TrimSpace(strings.Join(args[1:], " "))
	if body == "" {
		s.postSystem("usage: /sign @user <message>")
		return s, nil
	}
	if owner == me {
		s.postSystem("you can /post to your own profile — /sign is for friends' guestbooks")
		return s, nil
	}
	friends, err := s.data.AreFriends(me, owner)
	if err != nil {
		s.postSystem("/sign: " + err.Error())
		return s, nil
	}
	if !friends {
		s.postSystem(fmt.Sprintf("you can only sign a friend's guestbook — try /friend @%s first", owner))
		return s, nil
	}
	if _, err := s.data.SignGuestbook(owner, me, body); err != nil {
		s.postSystem("/sign: " + err.Error())
		return s, nil
	}
	s.postSystem(fmt.Sprintf("signed @%s's guestbook", owner))
	return s, nil
}
