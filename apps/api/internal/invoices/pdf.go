package invoices

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf/v2"

	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/promptpay"
	"github.com/odysight/crm/pkg/thaibaht"
)

// Sarabun (SIL OFL 1.1, see fonts/OFL.txt) covers Thai and Latin, so Thai
// customer names, addresses and the bilingual labels render correctly.
var (
	//go:embed fonts/Sarabun-Regular.ttf
	fontRegular []byte
	//go:embed fonts/Sarabun-Bold.ttf
	fontBold []byte
)

const fontFamily = "Sarabun"

// renderInvoicePDF produces a single-page A4 bilingual (Thai/English)
// invoice. A VAT-registered company gets a full tax invoice / receipt
// (ใบกำกับภาษี/ใบเสร็จรับเงิน) with both parties' tax IDs; otherwise a plain
// invoice (ใบแจ้งหนี้). When a PromptPay ID is configured, a QR for the net
// payable amount is printed so the customer can pay from any Thai bank app.
func renderInvoicePDF(inv Invoice, company settings.Company, pay settings.PaymentSettings) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes(fontFamily, "", fontRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", fontBold)
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	ink := func(r, g, b int) { pdf.SetTextColor(r, g, b) }
	font := func(style string, size float64) { pdf.SetFont(fontFamily, style, size) }
	const left, right, width = 15.0, 195.0, 180.0

	// Seller block (left).
	sellerName := strings.TrimSpace(company.LegalName)
	if sellerName == "" {
		sellerName = company.Name
	}
	pdf.SetXY(left, 15)
	font("B", 14)
	ink(30, 41, 59)
	pdf.MultiCell(100, 6.5, sellerName, "", "L", false)
	var seller []string
	if a := strings.TrimSpace(company.Address); a != "" {
		seller = append(seller, a)
	}
	if p := strings.TrimSpace(company.Phone); p != "" {
		seller = append(seller, "โทร / Tel: "+p)
	}
	if id := digitsOf(company.TaxID); id != "" {
		seller = append(seller, "เลขประจำตัวผู้เสียภาษี / Tax ID: "+formatTaxID(id), branchLabel(company.TaxBranch))
	}
	font("", 9.5)
	ink(71, 85, 105)
	pdf.SetX(left)
	pdf.MultiCell(100, 4.6, strings.Join(seller, "\n"), "", "L", false)
	sellerBottom := pdf.GetY()

	// Title + meta (right).
	thTitle, enTitle := "ใบแจ้งหนี้", "INVOICE"
	if company.VATRegistered {
		thTitle, enTitle = "ใบกำกับภาษี / ใบเสร็จรับเงิน", "TAX INVOICE / RECEIPT"
	}
	pdf.SetXY(115, 15)
	font("B", 15)
	ink(30, 58, 95)
	pdf.CellFormat(80, 7, thTitle, "", 2, "R", false, 0, "")
	font("B", 10)
	pdf.CellFormat(80, 5, enTitle, "", 2, "R", false, 0, "")
	pdf.Ln(2)
	font("", 9.5)
	ink(30, 41, 59)
	meta := [][2]string{
		{"เลขที่ / No.", inv.InvoiceNumber},
		{"วันที่ / Date", inv.IssuedAt.Format("02/01/2006")},
		{"อ้างอิง / Booking", inv.BookingNumber},
	}
	if inv.BillingPeriodStart != nil && inv.BillingPeriodEnd != nil {
		meta = append(meta, [2]string{"งวด / Period",
			inv.BillingPeriodStart.Format("02/01/2006") + " – " + inv.BillingPeriodEnd.Format("02/01/2006")})
	}
	for _, m := range meta {
		pdf.SetX(115)
		pdf.CellFormat(35, 5, m[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(45, 5, m[1], "", 1, "R", false, 0, "")
	}
	y := pdf.GetY()
	if sellerBottom > y {
		y = sellerBottom
	}

	// Customer box.
	y += 5
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetFillColor(248, 250, 252)
	var buyer []string
	buyer = append(buyer, inv.CustomerName)
	if a := strings.TrimSpace(inv.Address); a != "" {
		buyer = append(buyer, a)
	}
	if id := digitsOf(inv.CustomerTaxID); id != "" {
		buyer = append(buyer, "เลขประจำตัวผู้เสียภาษี / Tax ID: "+formatTaxID(id)+"  "+branchLabel(inv.CustomerTaxBranch))
	}
	boxH := 9 + float64(len(buyer))*5
	pdf.Rect(left, y, width, boxH, "FD")
	pdf.SetXY(left+3, y+2)
	font("B", 9.5)
	ink(100, 116, 139)
	pdf.CellFormat(0, 5, "ลูกค้า / Customer", "", 1, "L", false, 0, "")
	font("", 10)
	ink(30, 41, 59)
	pdf.SetX(left + 3)
	pdf.MultiCell(width-6, 5, strings.Join(buyer, "\n"), "", "L", false)
	y += boxH + 6

	// Line items.
	cols := []struct {
		label string
		w     float64
		align string
	}{
		{"ลำดับ\nNo.", 14, "C"},
		{"รายการ\nDescription", 86, "L"},
		{"จำนวน\nQty", 18, "C"},
		{"ราคาต่อหน่วย\nUnit price", 31, "R"},
		{"จำนวนเงิน\nAmount", 31, "R"},
	}
	pdf.SetFillColor(30, 58, 95)
	ink(255, 255, 255)
	font("B", 8.5)
	x := left
	for _, c := range cols {
		pdf.SetXY(x, y)
		pdf.MultiCell(c.w, 4.5, c.label, "", c.align, true)
		x += c.w
	}
	y += 9
	font("", 10)
	ink(30, 41, 59)
	row := []string{"1", inv.ServiceName, "1", amount(inv.Subtotal), amount(inv.Subtotal)}
	x = left
	for i, c := range cols {
		pdf.SetXY(x, y+1)
		pdf.MultiCell(c.w, 6, row[i], "", c.align, false)
		x += c.w
	}
	y += 14
	pdf.Line(left, y, right, y)

	// Totals (right) and amount in words (left).
	y += 3
	totals := [][2]string{{"รวมเป็นเงิน / Subtotal", amount(inv.Subtotal)}}
	if inv.TaxRate > 0 {
		totals = append(totals, [2]string{fmt.Sprintf("ภาษีมูลค่าเพิ่ม / VAT %s%%", trimRate(inv.TaxRate)), amount(inv.TaxAmount)})
	}
	totals = append(totals, [2]string{"จำนวนเงินรวมทั้งสิ้น / Grand total", amount(inv.Total)})
	if inv.WithholdingAmount > 0 {
		totals = append(totals,
			[2]string{fmt.Sprintf("หักภาษี ณ ที่จ่าย / WHT %s%%", trimRate(inv.WithholdingRate)), "-" + amount(inv.WithholdingAmount)},
			[2]string{"ยอดชำระสุทธิ / Net payable", amount(inv.NetPayable())})
	}
	ty := y
	for i, t := range totals {
		last := i == len(totals)-1
		if last {
			font("B", 11)
			pdf.SetFillColor(241, 245, 249)
		} else {
			font("", 10)
		}
		pdf.SetXY(110, ty)
		pdf.CellFormat(55, 7, t[0], "", 0, "L", last, 0, "")
		pdf.CellFormat(30, 7, t[1], "", 1, "R", last, 0, "")
		ty += 7
	}
	pdf.SetXY(left, y)
	font("B", 9)
	ink(100, 116, 139)
	pdf.CellFormat(90, 5, "จำนวนเงินตัวอักษร / Amount in words", "", 2, "L", false, 0, "")
	font("B", 10.5)
	ink(30, 41, 59)
	pdf.MultiCell(90, 5.5, "("+thaibaht.Text(inv.Total)+")", "", "L", false)
	if inv.Currency != "" && inv.Currency != "THB" {
		font("", 8.5)
		ink(100, 116, 139)
		pdf.SetX(left)
		pdf.MultiCell(90, 4.5, "Amounts in "+inv.Currency, "", "L", false)
	}
	y = ty + 6

	// Payment: PromptPay QR for the net payable, plus bank details.
	payTarget := strings.TrimSpace(pay.PromptPayID)
	bank := strings.TrimSpace(pay.BankAccount)
	if (payTarget != "" || bank != "") && inv.Status != StatusPaid && inv.Status != StatusVoid {
		font("B", 10)
		ink(30, 41, 59)
		pdf.SetXY(left, y)
		pdf.CellFormat(0, 6, "ช่องทางการชำระเงิน / Payment", "", 1, "L", false, 0, "")
		textX := left
		if payTarget != "" && (inv.Currency == "" || inv.Currency == "THB") {
			if payload, err := promptpay.Payload(payTarget, inv.NetPayable()); err == nil {
				if png, err := promptpay.PNG(payload, 360); err == nil {
					name := "promptpay-" + inv.InvoiceNumber
					pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
					pdf.ImageOptions(name, left, y+7, 34, 34, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
					textX = left + 38
				}
			}
		}
		pdf.SetXY(textX, y+8)
		font("", 9.5)
		ink(71, 85, 105)
		var lines []string
		if payTarget != "" {
			d, _ := promptpay.Normalize(payTarget)
			lines = append(lines, "สแกนเพื่อชำระด้วย PromptPay / Scan to pay with PromptPay",
				"PromptPay: "+d, "ยอดชำระ / Amount: "+inv.Currency+" "+amount(inv.NetPayable()))
		}
		if bank != "" {
			lines = append(lines, "", "โอนเงินเข้าบัญชี / Bank transfer:", bank)
		}
		pdf.MultiCell(right-textX, 4.8, strings.Join(lines, "\n"), "", "L", false)
		y += 45
	}

	// Paid stamp.
	if inv.Status == StatusPaid {
		pdf.SetDrawColor(22, 163, 74)
		ink(22, 163, 74)
		font("B", 16)
		pdf.SetLineWidth(0.8)
		pdf.Rect(140, y, 55, 14, "D")
		pdf.SetXY(140, y+3.5)
		pdf.CellFormat(55, 7, "ชำระแล้ว / PAID", "", 0, "C", false, 0, "")
		pdf.SetLineWidth(0.2)
		pdf.SetDrawColor(203, 213, 225)
	}

	// Signatures.
	sy := 245.0
	font("", 9.5)
	ink(71, 85, 105)
	for i, label := range []string{"ผู้รับเงิน / Collector", "ผู้มีอำนาจลงนาม / Authorized signature"} {
		sx := left + float64(i)*95
		pdf.Line(sx+5, sy, sx+80, sy)
		pdf.SetXY(sx, sy+1)
		pdf.CellFormat(85, 5, label, "", 2, "C", false, 0, "")
		pdf.SetX(sx)
		pdf.CellFormat(85, 5, "วันที่ / Date ____/____/______", "", 0, "C", false, 0, "")
	}

	footer := strings.TrimSpace(company.InvoiceFooter)
	if footer == "" {
		footer = "ขอบคุณที่ใช้บริการ / Thank you for your business!"
	}
	pdf.SetXY(left, 272)
	font("", 9)
	ink(148, 163, 184)
	pdf.MultiCell(width, 4.5, footer, "", "C", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gofpdf output: %w", err)
	}
	return buf.Bytes(), nil
}

func amount(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.2f", v)
	intPart, frac := s[:len(s)-3], s[len(s)-3:]
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String() + frac
	}
	return b.String() + frac
}

func trimRate(r float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", r), "0"), ".")
}

func digitsOf(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatTaxID prints a 13-digit TIN in the customary 1-2345-67890-12-3 form.
func formatTaxID(id string) string {
	if len(id) != 13 {
		return id
	}
	return id[0:1] + "-" + id[1:5] + "-" + id[5:10] + "-" + id[10:12] + "-" + id[12:13]
}

// branchLabel renders the branch the Revenue Department way: head office for
// 00000 (or blank), otherwise the numbered branch.
func branchLabel(branch string) string {
	b := digitsOf(branch)
	if b == "" || strings.Trim(b, "0") == "" {
		return "สำนักงานใหญ่ / Head office"
	}
	return "สาขาที่ / Branch " + strings.Repeat("0", max(0, 5-len(b))) + b
}
