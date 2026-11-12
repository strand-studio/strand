package clean

import (
	"strings"
	"unicode/utf8"
)

// ClipRunes shortens sanitized text to at most size characters like txt.Clip, without splitting a marker,
// for a column whose limit is measured in characters. Leading and trailing spaces are removed.
func ClipRunes(s string, size int) string {
	s = strings.TrimSpace(s)
	cut, runes := 0, 0

	for cut < len(s) {
		// A marker is ASCII, so its length in bytes is its length in characters.
		n, chars := markerLen(s[cut:]), 0

		if n > 0 {
			chars = n
		} else {
			_, n = utf8.DecodeRuneInString(s[cut:])
			chars = 1
		}

		if runes+chars > size {
			break
		}

		runes += chars
		cut += n
	}

	return strings.TrimSpace(s[:cut])
}

// shortenBytes shortens s to at most maxBytes without splitting a character or marker, ending it with suffix
// if it cuts.
func shortenBytes(s string, maxBytes int, suffix string) string {
	if len(s) <= maxBytes {
		return s
	}

	limit := maxBytes - len(suffix)

	if limit < 0 {
		return ""
	}

	cut := 0

	for cut < len(s) {
		n := markedUnitLen(s[cut:])

		if cut+n > limit {
			break
		}

		cut += n
	}

	return s[:cut] + suffix
}
