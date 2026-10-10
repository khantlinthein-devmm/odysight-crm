package sealbox

import (
	"bytes"
	"strings"
	"testing"
)

const testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func TestSealOpenRoundTrip(t *testing.T) {
	b, err := New(testKey)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("MA1234567")
	sealed, err := b.Seal(plain, []byte("ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, plain) {
		t.Fatal("sealed output must not contain the plaintext")
	}
	got, err := b.Open(sealed, []byte("ctx"))
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("round trip failed: %q %v", got, err)
	}
}

func TestOpenRejectsWrongContextKeyOrTamper(t *testing.T) {
	b, _ := New(testKey)
	sealed, _ := b.Seal([]byte("secret"), []byte("a"))
	if _, err := b.Open(sealed, []byte("b")); err == nil {
		t.Error("wrong associated data must fail")
	}
	other, _ := New(strings.Repeat("ff", 32))
	if _, err := other.Open(sealed, []byte("a")); err == nil {
		t.Error("wrong key must fail")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := b.Open(sealed, []byte("a")); err == nil {
		t.Error("tampered ciphertext must fail")
	}
	if _, err := b.Open([]byte("short"), nil); err == nil {
		t.Error("malformed input must fail")
	}
}

func TestNewRejectsBadKeys(t *testing.T) {
	for _, k := range []string{"", "abc", strings.Repeat("zz", 32), strings.Repeat("ab", 16)} {
		if _, err := New(k); err == nil {
			t.Errorf("key %q must be rejected", k)
		}
	}
}
