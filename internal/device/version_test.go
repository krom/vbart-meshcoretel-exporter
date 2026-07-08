package device

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{
			name:   "valid version string",
			raw:    "v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)",
			want:   "v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)",
			wantOK: true,
		},
		{
			name:   "trims surrounding whitespace",
			raw:    "\n  v1.16.0  \r\n",
			want:   "v1.16.0",
			wantOK: true,
		},
		{
			name:   "empty body",
			raw:    "",
			want:   "",
			wantOK: false,
		},
		{
			name:   "whitespace only",
			raw:    "   \n\t  ",
			want:   "",
			wantOK: false,
		},
		{
			name:   "oversized",
			raw:    strings.Repeat("a", MaxVersionLength+1),
			want:   "",
			wantOK: false,
		},
		{
			name:   "non-printable characters",
			raw:    "v1.16.0\x00\x01",
			want:   "",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseVersion([]byte(tc.raw))
			if ok != tc.wantOK {
				t.Fatalf("ParseVersion(%q) ok = %v, want %v", tc.raw, ok, tc.wantOK)
			}
			if got != tc.want {
				t.Errorf("ParseVersion(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
