// Package thaibaht spells an amount in Thai words the way tax invoices and
// receipts print it (the same convention as Excel's BAHTTEXT).
package thaibaht

import (
	"math"
	"strings"
)

var digitWords = [...]string{"ศูนย์", "หนึ่ง", "สอง", "สาม", "สี่", "ห้า", "หก", "เจ็ด", "แปด", "เก้า"}
var placeWords = [...]string{"", "สิบ", "ร้อย", "พัน", "หมื่น", "แสน"}

// Text returns e.g. "หนึ่งพันเจ็ดสิบบาทถ้วน" for 1070 or
// "ยี่สิบห้าสตางค์" for 0.25. Negative amounts are spelled as positive.
func Text(amount float64) string {
	satangTotal := int64(math.Round(math.Abs(amount) * 100))
	baht, satang := satangTotal/100, satangTotal%100
	switch {
	case baht == 0 && satang == 0:
		return "ศูนย์บาทถ้วน"
	case baht == 0:
		return number(satang) + "สตางค์"
	case satang == 0:
		return number(baht) + "บาทถ้วน"
	default:
		return number(baht) + "บาท" + number(satang) + "สตางค์"
	}
}

// number spells n > 0, grouping by millions (ล้าน).
func number(n int64) string {
	if n >= 1_000_000 {
		high, low := n/1_000_000, n%1_000_000
		s := number(high) + "ล้าน"
		if low > 0 {
			s += group(low)
		}
		return s
	}
	return group(n)
}

// group spells 0 < n < 1,000,000.
func group(n int64) string {
	var b strings.Builder
	d := make([]int64, 6)
	for i := 0; i < 6; i++ {
		d[i] = n % 10
		n /= 10
	}
	for pos := 5; pos >= 0; pos-- {
		v := d[pos]
		if v == 0 {
			continue
		}
		switch {
		case pos == 1 && v == 1:
			b.WriteString("สิบ")
			continue
		case pos == 1 && v == 2:
			b.WriteString("ยี่สิบ")
			continue
		case pos == 0 && v == 1 && hasHigher(d):
			b.WriteString("เอ็ด")
			continue
		}
		b.WriteString(digitWords[v])
		b.WriteString(placeWords[pos])
	}
	return b.String()
}

func hasHigher(d []int64) bool {
	for _, v := range d[1:] {
		if v != 0 {
			return true
		}
	}
	return false
}
