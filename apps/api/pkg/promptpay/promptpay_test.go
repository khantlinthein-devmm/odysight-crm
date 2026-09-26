package promptpay

import "testing"

// CRC-16/CCITT-FALSE standard check value.
func TestCRC16CheckValue(t *testing.T) {
	if got := crc16("123456789"); got != 0x29B1 {
		t.Fatalf("crc16 check = %04X, want 29B1", got)
	}
}

// CRC cross-checked with Python's binascii.crc_hqx(data, 0xFFFF).
func TestPayloadPhoneNoAmount(t *testing.T) {
	got, err := Payload("081-234-5678", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "00020101021129370016A000000677010111011300668123456785802TH530376463045D82"
	if got != want {
		t.Fatalf("payload\n got %s\nwant %s", got, want)
	}
}

func TestPayloadTaxIDWithAmount(t *testing.T) {
	got, err := Payload("1234567890123", 1070)
	if err != nil {
		t.Fatal(err)
	}
	prefix := "00020101021229370016A000000677010111021312345678901235802TH53037645407" + "1070.00" + "6304"
	if len(got) != len(prefix)+4 || got[:len(prefix)] != prefix {
		t.Fatalf("payload %s does not start with %s", got, prefix)
	}
	if crc := got[len(prefix):]; crc != crcHex(prefix) {
		t.Fatalf("crc %s, want %s", crc, crcHex(prefix))
	}
}

func crcHex(s string) string {
	const hex = "0123456789ABCDEF"
	c := crc16(s)
	return string([]byte{hex[c>>12&0xF], hex[c>>8&0xF], hex[c>>4&0xF], hex[c&0xF]})
}

func TestPayloadRejectsBadTarget(t *testing.T) {
	if _, err := Payload("12345", 10); err == nil {
		t.Fatal("expected error for a 5-digit target")
	}
}

func TestPNG(t *testing.T) {
	p, _ := Payload("0812345678", 99.5)
	img, err := PNG(p, 256)
	if err != nil || len(img) < 100 || string(img[1:4]) != "PNG" {
		t.Fatalf("png: err=%v len=%d", err, len(img))
	}
}
