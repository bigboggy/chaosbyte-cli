package capability

import (
	"fmt"
	"strings"

	"github.com/biscuit-auth/biscuit-go/v2/parser"
)

// VerifyContext carries the per-event facts the verifier evaluates
// against the token's checks. Room is the team slug; Action is one of
// the action_class values ("read", "edit", "refactor", "delete", "run",
// "commit", "review").
type VerifyContext struct {
	Room   string
	Action string
}

// Verify confirms the token authorizes the action in the context.
// Returns the parsed Token for callers that want to inspect facts.
// A nil or empty raw token returns ErrNoToken; the broker's caller
// decides whether nil is acceptable for the current phase.
func (i *Issuer) Verify(raw []byte, ctx VerifyContext) (Token, error) {
	if len(raw) == 0 {
		return Token{}, ErrNoToken
	}
	tok, err := ParseToken(raw)
	if err != nil {
		return Token{}, err
	}
	authorizer, err := tok.parsed.Authorizer(i.rootPub)
	if err != nil {
		return Token{}, fmt.Errorf("capability: authorizer: %w", err)
	}
	query := fmt.Sprintf(`
		room(%q);
		action_class(%q);
		allow if user($u);
	`, ctx.Room, ctx.Action)
	parsed, err := parser.FromStringAuthorizerWithParams(query, nil)
	if err != nil {
		return Token{}, fmt.Errorf("capability: parse query: %w", err)
	}
	authorizer.AddAuthorizer(parsed)
	if err := authorizer.Authorize(); err != nil {
		return Token{}, fmt.Errorf("capability: authorize: %w", err)
	}
	return tok, nil
}

// ErrNoToken is returned by Verify when the raw input is nil or empty.
// Phase 1 callers may treat this as "ok, no capability proof attached
// yet"; Phase 5 callers reject it.
var ErrNoToken = errNoToken{}

type errNoToken struct{}

func (errNoToken) Error() string { return "capability: no token provided" }

// ActionFromKind maps an event kind discriminator to the action_class
// the token must permit. The mapping is intentionally narrow in Phase
// 1; new kinds in later phases extend this table.
func ActionFromKind(kind string) string {
	switch {
	case kind == "chat.posted":
		return "post"
	case kind == "presence.joined", kind == "presence.left":
		return "read"
	case kind == "mod.tagged":
		return "review"
	case strings.HasPrefix(kind, "editor."):
		return "edit"
	case strings.HasPrefix(kind, "repo."):
		return "commit"
	case strings.HasPrefix(kind, "sandbox."), strings.HasPrefix(kind, "pty."):
		return "run"
	default:
		return "read"
	}
}
