package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestThemesRegistryHasBothPalettes locks the two shipped themes in place
// so the /themes command always has something to switch between. If a
// theme name changes, the lobby command needs to know.
func TestThemesRegistryHasBothPalettes(t *testing.T) {
	for _, name := range []string{"boggy", "workshop"} {
		if _, ok := Themes[name]; !ok {
			t.Fatalf("Themes map missing %q", name)
		}
	}
}

// TestApplyByNameSwitchesActive confirms a registered theme can be applied
// and that Active reflects the switch. This is what /themes depends on
// for the asterisk marker in its listing.
func TestApplyByNameSwitchesActive(t *testing.T) {
	before := Active
	t.Cleanup(func() { ApplyByName(before) })

	if !ApplyByName("workshop") {
		t.Fatal("ApplyByName(\"workshop\") returned false")
	}
	if Active != "workshop" {
		t.Errorf("Active = %q after switching to workshop, want %q", Active, "workshop")
	}
	if Bg != lipgloss.Color("#0a0a0c") {
		t.Errorf("Bg = %q after switch to workshop, want %q", Bg, "#0a0a0c")
	}

	if !ApplyByName("boggy") {
		t.Fatal("ApplyByName(\"boggy\") returned false")
	}
	if Active != "boggy" {
		t.Errorf("Active = %q after switching back, want %q", Active, "boggy")
	}
}

// TestApplyByNameUnknown confirms unknown names return false without
// mutating Active, so /themes can post a clean error.
func TestApplyByNameUnknown(t *testing.T) {
	before := Active
	t.Cleanup(func() { ApplyByName(before) })

	if ApplyByName("nope") {
		t.Fatal("ApplyByName(\"nope\") returned true, want false")
	}
	if Active != before {
		t.Errorf("Active changed to %q on unknown theme; want %q", Active, before)
	}
}

// TestListThemesSorted confirms the listing is in stable sorted order so
// /themes prints the same thing every time.
func TestListThemesSorted(t *testing.T) {
	got := ListThemes()
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("ListThemes not sorted: %v", got)
			break
		}
	}
	if len(got) < 2 {
		t.Errorf("ListThemes len = %d, want at least 2", len(got))
	}
}
