package invoices

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/odysight/crm/internal/settings"
)

func sampleInvoice() Invoice {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	return Invoice{
		InvoiceNumber: "INV-2026-0042", BookingNumber: "BK-2026-0107",
		CustomerName: "บริษัท เอบีซี เรสเตอรองต์ จำกัด", Address: "99/1 ถนนสุขุมวิท แขวงคลองเตย เขตคลองเตย กรุงเทพฯ 10110",
		ServiceName: "ทำความสะอาดครัว / Kitchen deep cleaning", Subtotal: 10000, TaxRate: 7, TaxAmount: 700, Total: 10700,
		Currency: "THB", Status: StatusIssued, CustomerTaxID: "0105561234567", CustomerTaxBranch: "00001",
		WithholdingRate: 3, WithholdingAmount: 300, BillingPeriodStart: &start, BillingPeriodEnd: &end,
		IssuedAt: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC),
	}
}

func TestRenderTaxInvoicePDF(t *testing.T) {
	company := settings.Company{Name: "Smile Clean", LegalName: "บริษัท สไมล์ คลีน (ประเทศไทย) จำกัด",
		Address: "1 ถนนสีลม บางรัก กรุงเทพฯ 10500", Phone: "02-123-4567",
		TaxID: "0105560000001", TaxBranch: "00000", VATRegistered: true}
	pay := settings.PaymentSettings{PromptPayID: "0105560000001", BankAccount: "กสิกรไทย 123-4-56789-0 บจ. สไมล์ คลีน"}
	pdf, err := renderInvoicePDF(sampleInvoice(), company, pay)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") || len(pdf) < 5000 {
		t.Fatalf("not a PDF (%d bytes)", len(pdf))
	}
	if out := os.Getenv("INVOICE_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}
}

func TestRenderPlainInvoicePDFWithoutTaxSettings(t *testing.T) {
	inv := sampleInvoice()
	inv.CustomerTaxID, inv.WithholdingAmount, inv.WithholdingRate = "", 0, 0
	inv.Status = StatusPaid
	if _, err := renderInvoicePDF(inv, settings.Company{Name: "Smile Clean"}, settings.PaymentSettings{}); err != nil {
		t.Fatal(err)
	}
}

func TestNetPayable(t *testing.T) {
	if got := sampleInvoice().NetPayable(); got != 10400 {
		t.Fatalf("net payable = %v, want 10400", got)
	}
}

func TestAmountAndTaxIDFormatting(t *testing.T) {
	if got := amount(1234567.5); got != "1,234,567.50" {
		t.Errorf("amount = %s", got)
	}
	if got := formatTaxID("0105561234567"); got != "0-1055-61234-56-7" {
		t.Errorf("formatTaxID = %s", got)
	}
	if got := branchLabel("1"); got != "สาขาที่ / Branch 00001" {
		t.Errorf("branchLabel = %s", got)
	}
	if got := branchLabel(""); got != "สำนักงานใหญ่ / Head office" {
		t.Errorf("branchLabel blank = %s", got)
	}
}
