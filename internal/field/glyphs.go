package field

// Glyph atlas for the alphabet and digits, ported verbatim from the
// ertdfgcvb.xyz landing-page engine (Q dictionary in js.js). Each glyph is
// 8 wide by 11 tall, packed row-major as three int32 words per glyph. Bit i
// of word k encodes pixel i+32k of the 88-cell grid.

const (
	glyphWidth  = 8
	glyphHeight = 11
	glyphCells  = glyphWidth * glyphHeight
	wordsPer    = 3
	wordLen     = 3
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var glyphData = []int32{
	1667446300, 1667465059, 25443,
	1667457855, 1667457855, 16227,
	50553662, 50529027, 15971,
	1667445535, 1667457891, 7987,
	50529151, 50529087, 32515,
	50529151, 50529087, 771,
	50553662, 1667457915, 32355,
	1667457891, 1667457919, 25443,
	202116159, 202116108, 16140,
	808464504, 858796080, 7731,
	456352611, 857411343, 25443,
	50529027, 50529027, 32515,
	2138530659, 1667984235, 25443,
	1734828899, 1936948079, 25443,
	1667457854, 1667457891, 15971,
	1667457855, 50544483, 771,
	1667457854, 1868784483, 6305371,
	1667457855, 858987327, 25443,
	100885310, 1613764620, 15971,
	404232447, 404232216, 6168,
	1667457891, 1667457891, 15971,
	1667457891, 912483171, 2076,
	1801675619, 2137746283, 13878,
	912483171, 1664490524, 25443,
	858993459, 202120755, 3084,
	811622527, 50727960, 32515,
	1801675582, 1667984235, 15971,
	404626456, 404232216, 32280,
	1616929598, 101455920, 32515,
	1616929598, 1616928828, 15971,
	1010315296, 813642550, 12336,
	50529151, 1616928831, 15971,
	50529852, 1667457855, 15971,
	1616929663, 202119216, 3084,
	1667457854, 1667457854, 15971,
	-471604290, -522125597, 8429232,
}

var glyphCache map[rune][glyphCells]bool
var emptyGlyph [glyphCells]bool

func init() {
	glyphCache = make(map[rune][glyphCells]bool, len(alphabet))
	for i, r := range alphabet {
		off := i * wordsPer
		w0 := uint32(glyphData[off])
		w1 := uint32(glyphData[off+1])
		w2 := uint32(glyphData[off+2])
		var g [glyphCells]bool
		for b := 0; b < glyphCells; b++ {
			var word uint32
			switch b / 32 {
			case 0:
				word = w0
			case 1:
				word = w1
			default:
				word = w2
			}
			g[b] = (word>>uint(b%32))&1 != 0
		}
		glyphCache[r] = g
	}
}

func glyphFor(r rune) [glyphCells]bool {
	if g, ok := glyphCache[r]; ok {
		return g
	}
	return emptyGlyph
}

// composeWord lays a 3-letter word into a w*h bitmap row-major.
func composeWord(word string) ([]bool, int, int) {
	w := glyphWidth * wordLen
	h := glyphHeight
	bmp := make([]bool, w*h)
	runes := []rune(word)
	for i := 0; i < wordLen; i++ {
		var g [glyphCells]bool
		if i < len(runes) {
			g = glyphFor(runes[i])
		}
		for y := 0; y < glyphHeight; y++ {
			for x := 0; x < glyphWidth; x++ {
				bmp[y*w+i*glyphWidth+x] = g[y*glyphWidth+x]
			}
		}
	}
	return bmp, w, h
}
