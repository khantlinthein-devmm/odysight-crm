// Package webpush sends Web Push notifications (RFC 8030) with payload
// encryption (RFC 8291, aes128gcm) and VAPID authentication (RFC 8292),
// using only the standard library.
package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrGone means the push service no longer knows the subscription (the user
// unsubscribed or the browser dropped it); the caller should delete it.
var ErrGone = errors.New("push subscription is gone")

// Subscription is what the browser's PushManager.subscribe returns.
type Subscription struct {
	Endpoint string
	P256dh   string // base64url, uncompressed P-256 public key
	Auth     string // base64url, 16-byte auth secret
}

// Keys is the application server's VAPID key pair.
type Keys struct {
	private *ecdsa.PrivateKey
}

// GenerateKeys creates a new VAPID key pair.
func GenerateKeys() (Keys, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Keys{}, err
	}
	return Keys{private: k}, nil
}

// MarshalPrivate encodes the private key for storage (base64 of SEC 1 DER).
func (k Keys) MarshalPrivate() (string, error) {
	der, err := x509.MarshalECPrivateKey(k.private)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(der), nil
}

// ParseKeys decodes a key pair stored with MarshalPrivate.
func ParseKeys(stored string) (Keys, error) {
	der, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return Keys{}, fmt.Errorf("decode vapid key: %w", err)
	}
	k, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return Keys{}, fmt.Errorf("parse vapid key: %w", err)
	}
	return Keys{private: k}, nil
}

// PublicKey is the base64url uncompressed public key browsers need as
// applicationServerKey.
func (k Keys) PublicKey() string {
	pub, err := k.private.PublicKey.ECDH()
	if err != nil {
		return ""
	}
	return b64(pub.Bytes())
}

// Sender delivers notifications.
type Sender struct {
	Keys    Keys
	Subject string // "mailto:…" or "https://…" contact for the push service
	Client  *http.Client
}

// Send encrypts payload for sub and posts it to the push service. ttl is how
// long the service may hold the message for an offline device.
func (s *Sender) Send(ctx context.Context, sub Subscription, payload []byte, ttl time.Duration) error {
	body, err := Encrypt(sub, payload, nil, nil)
	if err != nil {
		return err
	}
	endpoint, err := url.Parse(sub.Endpoint)
	if err != nil || endpoint.Scheme != "https" {
		return fmt.Errorf("invalid push endpoint")
	}
	jwt, err := s.vapidJWT(endpoint.Scheme + "://" + endpoint.Host)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", strconv.Itoa(int(ttl.Seconds())))
	req.Header.Set("Urgency", "high")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+s.Keys.PublicKey())
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("push request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return ErrGone
	case resp.StatusCode >= 300:
		return fmt.Errorf("push service returned %d", resp.StatusCode)
	}
	return nil
}

// vapidJWT signs the ES256 token identifying this server to the push service.
func (s *Sender) vapidJWT(audience string) (string, error) {
	header := b64([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, _ := json.Marshal(map[string]any{
		"aud": audience,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": s.Subject,
	})
	signingInput := header + "." + b64(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, sig, err := ecdsa.Sign(rand.Reader, s.Keys.private, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign vapid jwt: %w", err)
	}
	raw := make([]byte, 64)
	r.FillBytes(raw[:32])
	sig.FillBytes(raw[32:])
	return signingInput + "." + b64(raw), nil
}

// Encrypt produces the aes128gcm message body (RFC 8291). salt and asKey are
// generated when nil; tests pass fixed values to reproduce known vectors.
func Encrypt(sub Subscription, plaintext []byte, salt []byte, asKey *ecdh.PrivateKey) ([]byte, error) {
	uaRaw, err := unb64(sub.P256dh)
	if err != nil {
		return nil, fmt.Errorf("bad p256dh: %w", err)
	}
	authSecret, err := unb64(sub.Auth)
	if err != nil || len(authSecret) != 16 {
		return nil, fmt.Errorf("bad auth secret")
	}
	uaPub, err := ecdh.P256().NewPublicKey(uaRaw)
	if err != nil {
		return nil, fmt.Errorf("bad p256dh: %w", err)
	}
	if asKey == nil {
		if asKey, err = ecdh.P256().GenerateKey(rand.Reader); err != nil {
			return nil, err
		}
	}
	if salt == nil {
		salt = make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, err
		}
	}
	shared, err := asKey.ECDH(uaPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}
	asPub := asKey.PublicKey().Bytes()

	prkKey, err := hkdf.Extract(sha256.New, shared, authSecret)
	if err != nil {
		return nil, err
	}
	keyInfo := "WebPush: info\x00" + string(uaRaw) + string(asPub)
	ikm, err := hkdf.Expand(sha256.New, prkKey, keyInfo, 32)
	if err != nil {
		return nil, err
	}
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Single record: plaintext followed by the 0x02 last-record delimiter.
	record := append(append([]byte{}, plaintext...), 0x02)
	ciphertext := gcm.Seal(nil, nonce, record, nil)

	const recordSize = 4096
	if len(ciphertext) > recordSize {
		return nil, fmt.Errorf("payload too large for one record")
	}
	var out bytes.Buffer
	out.Write(salt)
	_ = binary.Write(&out, binary.BigEndian, uint32(recordSize))
	out.WriteByte(byte(len(asPub)))
	out.Write(asPub)
	out.Write(ciphertext)
	return out.Bytes(), nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "=")
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	return base64.RawURLEncoding.DecodeString(s)
}
