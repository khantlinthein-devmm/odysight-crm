// Package promptpay builds Thai PromptPay QR payloads (EMVCo merchant-
// presented format, as issued by the Bank of Thailand) and renders them as
// PNG images.
package promptpay

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const aid = "A000000677010111"

// Normalize reduces a PromptPay target to its digits and reports whether it
// is a usable mobile number (10 digits), national/tax ID (13) or e-wallet
// ID (15).
func Normalize(target string) (string, bool) {
	var b strings.Builder
	for _, r := range target {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch len(d) {
	case 10, 13, 15:
		return d, true
	}
	return d, false
}

func field(id, value string) string {
	return fmt.Sprintf("%s%02d%s", id, len(value), value)
}

// Payload returns the QR string for target. amount > 0 makes a one-time QR
// that pre-fills the amount in the payer's banking app.
func Payload(target string, amount float64) (string, error) {
	d, ok := Normalize(target)
	if !ok {
		return "", fmt.Errorf("promptpay: %q is not a 10-digit phone, 13-digit tax ID or 15-digit e-wallet ID", target)
	}
	var acct string
	switch len(d) {
	case 10:
		// Mobile numbers are sent as 0066 + number without the leading 0.
		acct = field("01", "0066"+strings.TrimPrefix(d, "0"))
	case 13:
		acct = field("02", d)
	default:
		acct = field("03", d)
	}
	pointOfInit := "11"
	if amount > 0 {
		pointOfInit = "12"
	}
	p := field("00", "01") +
		field("01", pointOfInit) +
		field("29", field("00", aid)+acct) +
		field("58", "TH") +
		field("53", "764")
	if amount > 0 {
		p += field("54", fmt.Sprintf("%.2f", amount))
	}
	p += "6304"
	return p + fmt.Sprintf("%04X", crc16(p)), nil
}

// crc16 is CRC-16/CCITT-FALSE (poly 0x1021, init 0xFFFF), per EMVCo.
func crc16(s string) uint16 {
	crc := uint16(0xFFFF)
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// PNG renders the payload as a size×size QR image.
func PNG(payload string, size int) ([]byte, error) {
	code, err := qr.Encode(payload, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("promptpay qr: %w", err)
	}
	code, err = barcode.Scale(code, size, size)
	if err != nil {
		return nil, fmt.Errorf("promptpay qr scale: %w", err)
	}
	// Re-draw as 8-bit grayscale: some PDF writers reject 16-bit PNGs.
	gray := image.NewGray(code.Bounds())
	draw.Draw(gray, gray.Bounds(), code, code.Bounds().Min, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, gray); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
