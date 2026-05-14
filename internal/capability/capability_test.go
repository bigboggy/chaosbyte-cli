package capability

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bchayka/gitstatus/internal/identity"
)

func samplePrincipal() identity.Principal {
	return identity.Principal{
		ID:          "pk:test",
		DisplayName: "@test",
		Teams:       []string{"vibespace", "monobyte"},
		Roles:       []string{"operator"},
		SessionID:   uuid.New(),
		Kind:        identity.KindHuman,
	}
}

func newIssuer(t *testing.T) *Issuer {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "biscuit-root.key")
	issuer, err := NewIssuer(path)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return issuer
}

func TestIssueAndVerify(t *testing.T) {
	issuer := newIssuer(t)
	tok, err := issuer.IssueSession(samplePrincipal(), time.Hour)
	if err != nil {
		t.Fatalf("IssueSession: %v", err)
	}
	if tok.IsEmpty() {
		t.Fatal("token should not be empty")
	}

	_, err = issuer.Verify(tok.Bytes(), VerifyContext{Room: "vibespace", Action: "post"})
	if err != nil {
		t.Errorf("Verify for authorized room failed: %v", err)
	}
}

func TestVerifyRejectsWrongRoom(t *testing.T) {
	issuer := newIssuer(t)
	tok, err := issuer.IssueSession(samplePrincipal(), time.Hour)
	if err != nil {
		t.Fatalf("IssueSession: %v", err)
	}
	_, err = issuer.Verify(tok.Bytes(), VerifyContext{Room: "acme", Action: "post"})
	if err == nil {
		t.Error("Verify should have rejected unauthorized room")
	}
}

func TestVerifyNilTokenReturnsErrNoToken(t *testing.T) {
	issuer := newIssuer(t)
	_, err := issuer.Verify(nil, VerifyContext{Room: "vibespace", Action: "post"})
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("expected ErrNoToken, got %v", err)
	}
}

func TestNewIssuerPersistsRootKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "biscuit-root.key")

	i1, err := NewIssuer(path)
	if err != nil {
		t.Fatal(err)
	}
	pub1 := i1.RootPublicKey()

	// Re-open: same key.
	i2, err := NewIssuer(path)
	if err != nil {
		t.Fatal(err)
	}
	pub2 := i2.RootPublicKey()
	if string(pub1) != string(pub2) {
		t.Error("root key not stable across opens")
	}

	// File should be 0600.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perms = %o, want 0600", info.Mode().Perm())
	}
}

func TestActionFromKind(t *testing.T) {
	cases := []struct {
		kind   string
		action string
	}{
		{"chat.posted", "post"},
		{"presence.joined", "read"},
		{"presence.left", "read"},
		{"mod.tagged", "review"},
		{"editor.opened", "edit"},
		{"repo.committed", "commit"},
		{"sandbox.spawned", "run"},
		{"pty.chunk", "run"},
		{"unknown.kind", "read"},
	}
	for _, c := range cases {
		if got := ActionFromKind(c.kind); got != c.action {
			t.Errorf("ActionFromKind(%q) = %q, want %q", c.kind, got, c.action)
		}
	}
}
