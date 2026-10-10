package invoices

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/odysight/crm/internal/settings"
)

func sampleQuote(lines int) QuoteDoc {
	valid := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	q := QuoteDoc{
		Number: "Q-2026-0007", Date: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC), ValidUntil: &valid,
		Status: "draft", CustomerName: "บริษัท เอบีซี เรสเตอรองต์ จำกัด",
		Address: "99/1 ถนนสุขุมวิท แขวงคลองเตย กรุงเทพฯ 10110", TaxID: "0105561234567", TaxBranch: "00000",
		SiteName: "สาขาสุขุมวิท", TaxRate: 7, Currency: "THB",
		Notes: "รวมน้ำยาและอุปกรณ์ / Cleaning supplies included.",
	}
	for i := 0; i < lines; i++ {
		amt := float64(1500 * (i + 1))
		q.Lines = append(q.Lines, QuoteLine{
			Name: fmt.Sprintf("ทำความสะอาดพื้นที่ %d / Area %d cleaning", i+1, i+1), Description: "รายสัปดาห์ / Weekly",
			Quantity: float64(i + 1), UnitPrice: 1500, Amount: amt,
		})
		q.Subtotal += amt
	}
	q.Total = q.Subtotal * 1.07
	return q
}

func pageCount(pdf []byte) int { return bytes.Count(pdf, []byte("/Type /Page\n")) }

func TestRenderQuotePDF(t *testing.T) {
	company := settings.Company{Name: "Smile Clean", LegalName: "บริษัท สไมล์ คลีน (ประเทศไทย) จำกัด", TaxID: "0105560000001"}
	pdf, err := RenderQuotePDF(sampleQuote(3), company)
	if err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("QUOTE_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") || pageCount(pdf) != 1 {
		t.Fatalf("3-line quote: %d pages", pageCount(pdf))
	}
	if out := os.Getenv("QUOTE_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}
	// A typical quote (several two-line items, accepted) fits on one page.
	five := sampleQuote(6)
	five.Status = "accepted"
	if pdf, _ := RenderQuotePDF(five, company); pageCount(pdf) != 1 {
		t.Fatalf("6-line accepted quote: %d pages, want 1", pageCount(pdf))
	}
	long := sampleQuote(40)
	long.Status = "accepted"
	pdf, err = RenderQuotePDF(long, company)
	if err != nil {
		t.Fatal(err)
	}
	if pageCount(pdf) < 2 {
		t.Fatalf("40-line quote should continue on a second page, got %d", pageCount(pdf))
	}
	if out := os.Getenv("QUOTE_LONG_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}
}

func TestDepositOf(t *testing.T) {
	for _, c := range []struct{ total, want float64 }{{9630, 4815}, {6527, 3263.5}, {0.03, 0.02}} {
		if got := depositOf(c.total); got != c.want {
			t.Errorf("depositOf(%v) = %v, want %v", c.total, got, c.want)
		}
	}
}
