package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

func sampleKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sshKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh public key: %v", err)
	}
	authorizedLine := string(ssh.MarshalAuthorizedKey(sshKey))
	return pub, priv, authorizedLine
}

func TestIDFromPubkeyDeterministic(t *testing.T) {
	pub, _, _ := sampleKeyPair(t)
	id1 := IDFromPubkey(pub)
	id2 := IDFromPubkey(pub)
	if id1 != id2 {
		t.Fatalf("non-deterministic: %s != %s", id1, id2)
	}
	if len(id1) < 10 || id1[:3] != "pk:" {
		t.Errorf("unexpected format: %q", id1)
	}
}

func TestIDFromPubkeyDistinct(t *testing.T) {
	pubA, _, _ := sampleKeyPair(t)
	pubB, _, _ := sampleKeyPair(t)
	if IDFromPubkey(pubA) == IDFromPubkey(pubB) {
		t.Fatal("two different keys produced the same ID")
	}
}

func TestPrincipalMembership(t *testing.T) {
	p := Principal{Teams: []string{"vibespace", "monobyte"}, Roles: []string{"operator"}}
	if !p.IsMemberOf("vibespace") {
		t.Error("should be member of vibespace")
	}
	if p.IsMemberOf("acme") {
		t.Error("should not be member of acme")
	}
	if !p.HasRole("operator") {
		t.Error("should have operator role")
	}
	if p.HasRole("admin") {
		t.Error("should not have admin role")
	}
}

func TestAllowlistRoundtrip(t *testing.T) {
	_, _, lineA := sampleKeyPair(t)
	_, _, lineB := sampleKeyPair(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.toml")
	contents := `[[principals]]
display = "@daniel"
pubkey = ` + tomlString(lineA) + `
teams = ["vibespace", "monobyte"]
roles = ["operator"]

[[principals]]
display = "@boggy"
pubkey = ` + tomlString(lineB) + `
teams = ["vibespace"]
roles = ["operator"]
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	a, err := LoadAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAllowlist: %v", err)
	}
	if got := a.Count(); got != 2 {
		t.Errorf("Count = %d, want 2", got)
	}
}

func TestAllowlistLookup(t *testing.T) {
	pub, _, line := sampleKeyPair(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.toml")
	contents := `[[principals]]
display = "@daniel"
pubkey = ` + tomlString(line) + `
teams = ["vibespace"]
roles = ["operator"]
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	a, err := LoadAllowlist(path)
	if err != nil {
		t.Fatal(err)
	}

	entry, ok := a.Lookup(pub)
	if !ok {
		t.Fatal("Lookup failed for inserted key")
	}
	if entry.Display != "@daniel" {
		t.Errorf("Display = %q, want @daniel", entry.Display)
	}

	// A different key should not match.
	otherPub, _, _ := sampleKeyPair(t)
	if _, ok := a.Lookup(otherPub); ok {
		t.Error("Lookup of unauthorized key returned true")
	}
}

func TestAllowlistSkipsMalformed(t *testing.T) {
	_, _, line := sampleKeyPair(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.toml")
	contents := `[[principals]]
display = "@valid"
pubkey = ` + tomlString(line) + `
teams = ["vibespace"]

[[principals]]
display = ""
pubkey = "ssh-ed25519 not-a-real-key"
teams = []
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAllowlist(path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Count() != 1 {
		t.Errorf("Count = %d, want 1 after skipping malformed", a.Count())
	}
}

func TestLocalPrincipal(t *testing.T) {
	p := LocalPrincipal()
	if p.ID != "local:boggy" {
		t.Errorf("ID = %q", p.ID)
	}
	if p.DisplayName != "@boggy" {
		t.Errorf("DisplayName = %q", p.DisplayName)
	}
	if !p.IsMemberOf("vibespace") {
		t.Error("local should be a member of vibespace")
	}
	if p.SessionID == uuid.Nil {
		t.Error("SessionID should be assigned")
	}
}

// tomlString quotes a string for inclusion in TOML where embedded
// quotes / newlines would otherwise break parsing. The OpenSSH key
// format already contains a trailing newline from MarshalAuthorizedKey;
// strip it before wrapping.
func tomlString(s string) string {
	out := []byte{'"'}
	for _, b := range []byte(s) {
		if b == '\n' || b == '\r' {
			continue
		}
		if b == '"' || b == '\\' {
			out = append(out, '\\')
		}
		out = append(out, b)
	}
	out = append(out, '"')
	return string(out)
}
