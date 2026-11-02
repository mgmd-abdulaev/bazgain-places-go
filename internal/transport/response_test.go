package transport

import "testing"

func TestMatchesETag(t *testing.T) {
	tests := []struct {
		name, header, current string
		want                  bool
	}{
		{"empty", "", `"123"`, false},
		{"strong", `"123"`, `"123"`, true},
		{"unquoted", `123`, `"123"`, true},
		{"weak", `W/"123"`, `"123"`, true},
		{"list", `"111", W/"123", "222"`, `"123"`, true},
		{"wildcard", `*`, `"123"`, true},
		{"mismatch", `"124"`, `"123"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesETag(tt.header, tt.current); got != tt.want {
				t.Fatalf("matchesETag(%q, %q) = %v, want %v", tt.header, tt.current, got, tt.want)
			}
		})
	}
}

func TestValidIDMatchesNestJSVersionsOneThroughEight(t *testing.T) {
	for version := byte('1'); version <= '8'; version++ {
		raw := "abcdefab-cdef-" + string(version) + "abc-8abc-abcdefabcdef"
		got, err := validID(raw)
		if err != nil || got != raw {
			t.Fatalf("version %c: got %q err=%v", version, got, err)
		}
	}
	upper := "ABCDEFAB-CDEF-4ABC-8ABC-ABCDEFABCDEF"
	got, err := validID(upper)
	if err != nil || got != "abcdefab-cdef-4abc-8abc-abcdefabcdef" {
		t.Fatalf("uppercase canonicalization: got %q err=%v", got, err)
	}
}

func TestValidIDRejectsMalformedVersionAndVariant(t *testing.T) {
	for _, raw := range []string{
		"not-a-uuid",
		"abcdefab-cdef-0abc-8abc-abcdefabcdef",
		"abcdefab-cdef-9abc-8abc-abcdefabcdef",
		"abcdefab-cdef-4abc-7abc-abcdefabcdef",
	} {
		if _, err := validID(raw); err == nil {
			t.Fatalf("validID(%q) expected an error", raw)
		}
	}
}
