package resources

import "strings"

// searchResults returns matches of query against the combined trending+top
// skill sets. Empty query returns the combined list unfiltered.
func (s *Screen) searchResults() []Skill {
	combined := append(append([]Skill{}, s.trending...), s.top...)
	if s.query == "" {
		return combined
	}
	q := strings.ToLower(s.query)
	var out []Skill
	for _, sk := range combined {
		if strings.Contains(strings.ToLower(sk.Name), q) ||
			strings.Contains(strings.ToLower(sk.Category), q) ||
			strings.Contains(strings.ToLower(sk.Description), q) {
			out = append(out, sk)
		}
	}
	return out
}
