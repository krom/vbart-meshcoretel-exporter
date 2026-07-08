package device

import (
	"strings"
	"unicode"
)

// MaxVersionLength bounds the accepted length of a "ver" command response,
// guarding against a misbehaving device flooding a Prometheus label value.
const MaxVersionLength = 256

// ParseVersion validates and trims a raw "ver" command response. It returns
// the cleaned version string and true if usable, or an empty string and
// false if the response is empty, whitespace-only, too long, or contains
// non-printable characters.
func ParseVersion(raw []byte) (string, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" || len(s) > MaxVersionLength {
		return "", false
	}
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return "", false
		}
	}
	return s, true
}
