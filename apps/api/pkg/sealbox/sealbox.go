// Package sealbox encrypts small blobs (scanned documents, ID numbers) at
// rest with AES-256-GCM. Sealed output is versioned so the format can evolve:
//
//	"SB1" | 12-byte nonce | ciphertext+tag
package sealbox

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var magic = []byte("SB1")

// ErrMalformed is returned for input that was not produced by Seal or was
// sealed with a different key.
var ErrMalformed = errors.New("sealbox: malformed or tampered data")

type Box struct {
	aead cipher.AEAD
}

// New builds a Box from a 32-byte key given as 64 hex characters
// (generate with `openssl rand -hex 32`).
func New(hexKey string) (*Box, error) {
	key, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("sealbox: key must be 64 hex characters (32 bytes)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sealbox: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sealbox: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Seal encrypts plain. aad binds the ciphertext to its context (e.g. the
// stored file name) so blobs cannot be swapped between records.
func (b *Box) Seal(plain, aad []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("sealbox: nonce: %w", err)
	}
	out := make([]byte, 0, len(magic)+len(nonce)+len(plain)+b.aead.Overhead())
	out = append(out, magic...)
	out = append(out, nonce...)
	return b.aead.Seal(out, nonce, plain, aad), nil
}

func (b *Box) Open(sealed, aad []byte) ([]byte, error) {
	ns := b.aead.NonceSize()
	if len(sealed) < len(magic)+ns+b.aead.Overhead() || !bytes.HasPrefix(sealed, magic) {
		return nil, ErrMalformed
	}
	nonce := sealed[len(magic) : len(magic)+ns]
	plain, err := b.aead.Open(nil, nonce, sealed[len(magic)+ns:], aad)
	if err != nil {
		return nil, ErrMalformed
	}
	return plain, nil
}
