package discussions

import "sort"

// flatComment is a single entry in the depth-first comment walk used when
// rendering the threaded comment tree as a flat list.
type flatComment struct {
	depth int
	c     *Comment
}

// flattenComments walks the comment tree depth-first, sorted by like-count at
// each level, returning entries paired with their depth.
func flattenComments(comments []Comment, depth int) []flatComment {
	idxs := make([]int, len(comments))
	for i := range idxs {
		idxs[i] = i
	}
	sort.SliceStable(idxs, func(i, j int) bool {
		return comments[idxs[i]].Likes > comments[idxs[j]].Likes
	})
	var out []flatComment
	for _, i := range idxs {
		c := &comments[i]
		out = append(out, flatComment{depth: depth, c: c})
		out = append(out, flattenComments(c.Comments, depth+1)...)
	}
	return out
}

func toggleLike(liked *bool, likes *int) {
	if *liked {
		*liked = false
		*likes--
	} else {
		*liked = true
		*likes++
	}
}
