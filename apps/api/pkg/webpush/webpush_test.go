package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// TestEncryptRFC8291Vector encrypts the RFC 8291 Appendix A example and
// decrypts it with the RFC's user-agent key. (The output was also checked
// against an independent Python implementation of the receiver.)
func TestEncryptRFC8291Vector(t *testing.T) {
	sub := Subscription{
		P256dh: "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4",
		Auth:   "BTBZMqHH6r4Tts7J_aSIgg",
	}
	asPriv, _ := base64.RawURLEncoding.DecodeString("yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw")
	asKey, err := ecdh.P256().NewPrivateKey(asPriv)
	if err != nil {
		t.Fatal(err)
	}
	salt, _ := base64.RawURLEncoding.DecodeString("DGv6ra1nlYgDCS1FRnbzlw")
	body, err := Encrypt(sub, []byte("When I grow up, I want to be a watermelon"), salt, asKey)
	if err != nil {
		t.Fatal(err)
	}
	// Decrypt as the browser would, with the RFC's user-agent private key.
	uaPriv, _ := base64.RawURLEncoding.DecodeString("q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94")
	ua, err := ecdh.P256().NewPrivateKey(uaPriv)
	if err != nil {
		t.Fatal(err)
	}
	if got := base64.RawURLEncoding.EncodeToString(ua.PublicKey().Bytes()); got != sub.P256dh {
		t.Fatalf("test key mismatch: %s", got)
	}
	if string(body[:16]) != string(salt) || body[20] != 65 || string(body[21:86]) != string(asKey.PublicKey().Bytes()) {
		t.Fatal("header must be salt || rs || idlen || as_public")
	}
	shared, _ := ua.ECDH(asKey.PublicKey())
	auth, _ := base64.RawURLEncoding.DecodeString(sub.Auth)
	prkKey, _ := hkdf.Extract(sha256.New, shared, auth)
	ikm, _ := hkdf.Expand(sha256.New, prkKey, "WebPush: info\x00"+string(ua.PublicKey().Bytes())+string(asKey.PublicKey().Bytes()), 32)
	prk, _ := hkdf.Extract(sha256.New, ikm, salt)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, body[86:], nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != "When I grow up, I want to be a watermelon\x02" {
		t.Fatalf("plaintext = %q", plain)
	}
}

func TestKeysRoundTripAndJWT(t *testing.T) {
	k, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	stored, err := k.MarshalPrivate()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := ParseKeys(stored)
	if err != nil || k2.PublicKey() != k.PublicKey() || len(k.PublicKey()) != 87 {
		t.Fatalf("round trip: %v %q", err, k2.PublicKey())
	}
	s := Sender{Keys: k, Subject: "mailto:ops@example.com"}
	jwt, err := s.vapidJWT("https://fcm.googleapis.com")
	if err != nil || strings.Count(jwt, ".") != 2 {
		t.Fatalf("jwt: %v %q", err, jwt)
	}
}
