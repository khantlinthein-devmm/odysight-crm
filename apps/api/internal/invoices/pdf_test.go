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
	pdf, err := renderInvoicePDF(sampleInvoice(), company, pay, 14, 0)
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
	if _, err := renderInvoicePDF(inv, settings.Company{Name: "Smile Clean"}, settings.PaymentSettings{}, 0, 0); err != nil {
		t.Fatal(err)
	}
}

func TestThaiDateAndReceiptMethods(t *testing.T) {
	if got := thaiDate(time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)); got != "10/07/2569" {
		t.Errorf("thaiDate = %s", got)
	}
	pay := settings.PaymentSettings{BankAccount: "KBank 063-1-50243-1"}
	m := receiptMethods(Receipt{Method: "bank_transfer", Reference: "slip 12"}, pay)
	if !m[1].checked || m[0].checked || m[2].checked || m[3].checked {
		t.Errorf("bank transfer ticks %+v", m)
	}
	if !strings.Contains(m[1].label, "063-1-50243-1") || !strings.Contains(m[1].label, "slip 12") {
		t.Errorf("transfer label = %q", m[1].label)
	}
	if m := receiptMethods(Receipt{Method: "line_pay"}, pay); !m[3].checked || !strings.Contains(m[3].label, "LINE Pay") {
		t.Errorf("other method = %+v", m[3])
	}
}

func TestSplitVAT(t *testing.T) {
	// ราคารวม VAT: 11,000 is 10,280.37 + 719.63, not 10,230 + 770.
	if sub, vat, total := splitVAT(11000, 7, true); sub != 10280.37 || vat != 719.63 || total != 11000 {
		t.Errorf("inclusive: %v + %v = %v", sub, vat, total)
	}
	if sub, vat, total := splitVAT(11000, 7, false); sub != 11000 || vat != 770 || total != 11770 {
		t.Errorf("exclusive: %v + %v = %v", sub, vat, total)
	}
	if sub, vat, total := splitVAT(11000, 0, true); sub != 11000 || vat != 0 || total != 11000 {
		t.Errorf("no VAT: %v + %v = %v", sub, vat, total)
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
	if got := formatTaxID("0105561234567"); got != "0 1055 61234 56 7" {
		t.Errorf("formatTaxID = %s", got)
	}
	if got := branchLabel("1"); got != "สาขาที่ / Branch 00001" {
		t.Errorf("branchLabel = %s", got)
	}
	if got := branchLabel(""); got != "สำนักงานใหญ่ / Head office" {
		t.Errorf("branchLabel blank = %s", got)
	}
}

func TestRenderDepositRequest(t *testing.T) {
	inv := sampleInvoice()
	inv.Status, inv.AmountPaid = StatusIssued, 0
	pay := settings.PaymentSettings{PromptPayID: "0812345678", BankAccount: "KBank 123-4-56789-0"}
	pdf, err := renderInvoicePDF(inv, settings.Company{Name: "Smile Clean"}, pay, 7, DepositPercent)
	if err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("DEPOSIT_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatal("not a PDF")
	}
}
