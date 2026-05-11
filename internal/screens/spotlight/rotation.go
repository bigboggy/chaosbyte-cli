package spotlight

import (
	"fmt"
	"time"
)

// RotateSeconds is how long each spotlight stays featured. Rotation is a pure
// function of wall-clock time so it stays consistent across renders.
const RotateSeconds = 300

// rotation returns the active spotlight index and seconds remaining until the
// next rotation.
func (s *Screen) rotation() (idx, secsRemaining int) {
	if len(s.items) == 0 {
		return 0, RotateSeconds
	}
	secs := time.Now().Unix()
	idx = int((secs / RotateSeconds) % int64(len(s.items)))
	secsRemaining = RotateSeconds - int(secs%RotateSeconds)
	return idx, secsRemaining
}

func mmss(secs int) string {
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}
