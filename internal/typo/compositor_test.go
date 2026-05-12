package typo

import (
	"strings"
	"testing"
	"time"
)

// TestCompositorDrawsLayoutAtPosition asserts a basic DrawLayout call writes
// the message body cells into the compositor grid at the expected origin.
func TestCompositorDrawsLayoutAtPosition(t *testing.T) {
	c := NewCompositor(40, 5)
	l := Prepare("msg-1", "hello", 80)
	s := NewState()
	c.DrawLayout(l, &s, nil, 2, 1, time.Now())

	out := stripANSI(c.Render())
	rows := strings.Split(out, "\n")
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[1][2:], "hello") {
		t.Errorf("row 1 should have 'hello' at col 2, got %q", rows[1])
	}
}

// TestCompositorSkipsBorrowedCells asserts that cells listed as borrowed
// don't get drawn at their natural position by DrawLayout.
func TestCompositorSkipsBorrowedCells(t *testing.T) {
	c := NewCompositor(40, 5)
	l := Prepare("msg-1", "hello", 80)
	s := NewState()
	borrowed := map[int]bool{0: true, 4: true} // skip 'h' and 'o'
	c.DrawLayout(l, &s, borrowed, 0, 0, time.Now())

	out := stripANSI(c.Render())
	rows := strings.Split(out, "\n")
	row0 := rows[0]
	if row0[0] == 'h' || row0[4] == 'o' {
		t.Errorf("borrowed cells leaked through: %q", row0)
	}
	if row0[1] != 'e' || row0[2] != 'l' || row0[3] != 'l' {
		t.Errorf("non-borrowed cells missing: %q", row0)
	}
}

// TestCompositorDrawTransformsDisplacement asserts that transforms appear at
// (natural + offset) positions.
func TestCompositorDrawTransformsDisplacement(t *testing.T) {
	c := NewCompositor(40, 6)
	l := Prepare("msg-1", "ab", 80)

	transforms := []CellTransform{
		{
			SourceLayoutID: "msg-1",
			SourceCellIdx:  0, // 'a' at natural (0, 0)
			OffsetX:        10,
			OffsetY:        2,
			EffectKind:     "pluck",
		},
		{
			SourceLayoutID: "msg-1",
			SourceCellIdx:  1, // 'b' at natural (1, 0)
			OffsetX:        -1, // would clip to negative; should be dropped
			OffsetY:        3,
		},
	}
	origins := map[string]LayoutOrigin{
		"msg-1": {Layout: l, X: 5, Y: 0},
	}
	c.DrawTransforms(transforms, origins, time.Now())

	out := stripANSI(c.Render())
	rows := strings.Split(out, "\n")
	// 'a' was at (5+0+10, 0+0+2) = (15, 2)
	if rows[2][15] != 'a' {
		t.Errorf("expected 'a' at (15, 2); got %q at col 15", string(rows[2][15]))
	}
	// 'b' was at (5+1+(-1), 0+0+3) = (5, 3)
	if rows[3][5] != 'b' {
		t.Errorf("expected 'b' at (5, 3); got %q at col 5", string(rows[3][5]))
	}
}

// TestIndexTransforms asserts that IndexTransforms groups by SourceLayoutID
// so the renderer can quickly skip-check borrowed cells.
func TestIndexTransforms(t *testing.T) {
	transforms := []CellTransform{
		{SourceLayoutID: "a", SourceCellIdx: 0},
		{SourceLayoutID: "a", SourceCellIdx: 3},
		{SourceLayoutID: "b", SourceCellIdx: 1},
	}
	idx := IndexTransforms(transforms)
	if !idx["a"][0] || !idx["a"][3] {
		t.Error("missing 'a' indices")
	}
	if !idx["b"][1] {
		t.Error("missing 'b' index 1")
	}
	if idx["b"][0] {
		t.Error("phantom 'b' index 0")
	}
}
