package totp

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// RFC 6238 Appendix B, SHA-1 column (8 digits), secret "12345678901234567890".
func TestRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{
		59:          "94287082",
		1111111109:  "07081804",
		1111111111:  "14050471",
		1234567890:  "89005924",
		2000000000:  "69279037",
		20000000000: "65353130",
	}
	for unix, want := range cases {
		got, err := CodeAt(secret, unix/Period, 8)
		if err != nil || got != want {
			t.Errorf("t=%d: got %s, want %s (%v)", unix, got, want, err)
		}
	}
}

func TestVerifyWindowAndFormat(t *testing.T) {
	secret, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	code, _ := CodeAt(secret, Step(now), Digits)
	if s, ok := Verify(secret, code, now); !ok || s != Step(now) {
		t.Fatal("current code rejected")
	}
	prev, _ := CodeAt(secret, Step(now)-1, Digits)
	if _, ok := Verify(secret, prev, now); !ok {
		t.Fatal("previous step (clock drift) rejected")
	}
	old, _ := CodeAt(secret, Step(now)-3, Digits)
	if _, ok := Verify(secret, old, now); ok {
		t.Fatal("code from 90 s ago accepted")
	}
	if _, ok := Verify(secret, code[:3]+" "+code[3:], now); !ok {
		t.Fatal("spaced code rejected")
	}
	if _, ok := Verify(secret, "12345", now); ok {
		t.Fatal("short code accepted")
	}
	u := URI("Smile Clean", "owner@example.com", secret)
	if !strings.HasPrefix(u, "otpauth://totp/Smile%20Clean:owner@example.com?") || !strings.Contains(u, "secret="+secret) {
		t.Fatalf("uri = %s", u)
	}
}
