package spotlight

import "fmt"

// mmss formats a seconds count as "M:SS" for the header / status line.
func mmss(secs int) string {
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}
