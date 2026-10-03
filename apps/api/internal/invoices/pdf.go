package invoices

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strings"
	"time"

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
	//go:embed assets/logo.png
	logoPNG []byte
)

const fontFamily = "Sarabun"

// docSpec describes one printed document; invoices and receipts share the
// layout (seller header, customer box, one line item, totals, signatures).
type docSpec struct {
	thTitle, enTitle string
	meta             [][2]string
	buyerName        string
	buyerAddress     string
	buyerTaxID       string
	buyerBranch      string
	description      string
	lineAmount       float64
	// items, when set, replaces the single description/lineAmount row.
	items    []docItem
	totals   [][2]string
	words    float64
	currency string
	// Payment block (QR + bank details); skipped when payAmount is zero.
	payAmount float64
	imageKey  string
	stamp     string
	stampRGB  [3]int
	// note is printed under the amount in words (e.g. how it was paid).
	note string
	// Thai-style signature blocks, left and right: the role printed under
	// the line, and on whose behalf ("seller", "buyer" or "") they sign.
	signLabels [2]string
	signFor    [2]string
	// signDate (the document date) is printed on the date line of the
	// blocks marked in signDated — our side; the customer's stays blank.
	signDate  time.Time
	signDated [2]bool
	// terms is a highlighted condition printed under the totals (e.g. the
	// deposit required to book, on quotations).
	terms string
	// copyLabel marks a single-copy document; copies prints the document
	// once per label (ต้นฉบับ / สำเนา).
	copyLabel string
	copies    []string
	// unitPriceCol adds the unit-price column (quotations); invoices and
	// receipts print one job per row: No. / Description / Qty / Amount.
	unitPriceCol bool
	// qtyUnit is printed after the quantity ("1 งาน").
	qtyUnit string
	// payMethods is the receipt's "paid by" checklist.
	payMethods []checkItem
}

// checkItem is one ticked or empty box with its label.
type checkItem struct {
	label   string
	checked bool
}

// thaiDate prints a date the Thai way, day/month/Buddhist-era year
// (10/07/2569).
func thaiDate(t time.Time) string {
	return fmt.Sprintf("%02d/%02d/%d", t.Day(), int(t.Month()), t.Year()+543)
}

// siteLine is the work-site line under a description.
func siteLine(site string) string {
	if site = strings.TrimSpace(site); site != "" {
		return "\nสถานที่ปฏิบัติงาน / Site: " + site
	}
	return ""
}

// renderInvoicePDF produces a bilingual (Thai/English) invoice — ใบแจ้งหนี้,
// the request for payment. VAT is shown when the invoice carries it; the tax
// invoice itself is the receipt issued once money is received. While money
// is still owed, a PromptPay QR for the balance is printed.
// dueDays is the payment term: the invoice is due that many days after it
// is issued (the same period after which overdue reminders go out).
// depositPct > 0 renders a deposit request: the PromptPay QR and the amount
// to pay are the deposit, not the full balance (unpaid invoices only).
func renderInvoicePDF(inv Invoice, company settings.Company, pay settings.PaymentSettings, dueDays, depositPct int) ([]byte, error) {
	if dueDays < 1 {
		dueDays = settings.DefaultOverdueReminderDays
	}
	meta := [][2]string{
		{"เลขที่ / No.", inv.InvoiceNumber},
		{"วันที่ / Date", thaiDate(inv.IssuedAt)},
		{"ครบกำหนด / Due date", thaiDate(inv.IssuedAt.AddDate(0, 0, dueDays))},
		{"อ้างอิง / Booking", inv.BookingNumber},
	}
	if inv.BillingPeriodStart != nil && inv.BillingPeriodEnd != nil {
		meta = append(meta, [2]string{"งวด / Period",
			thaiDate(*inv.BillingPeriodStart) + " – " + thaiDate(*inv.BillingPeriodEnd)})
	}
	totals := [][2]string{{"รวมเงิน / Total amount", amount(inv.Subtotal)}}
	if inv.TaxAmount > 0 {
		totals = append(totals, [2]string{fmt.Sprintf("ภาษีมูลค่าเพิ่ม / VAT %s%%", trimRate(inv.TaxRate)), amount(inv.TaxAmount)})
	}
	totals = append(totals, [2]string{"จำนวนเงินรวมทั้งสิ้น / Grand total", amount(inv.Total)})
	if inv.WithholdingAmount > 0 {
		totals = append(totals,
			[2]string{fmt.Sprintf("ภาษีหัก ณ ที่จ่าย / WHT %s%%", trimRate(inv.WithholdingRate)), "-" + amount(inv.WithholdingAmount)},
			[2]string{"ยอดเงินสุทธิ / Total net", amount(inv.NetPayable())})
	}
	if inv.AmountPaid > 0 && inv.Status != StatusPaid {
		totals = append(totals,
			[2]string{"ชำระแล้ว / Paid", "-" + amount(inv.AmountPaid)},
			[2]string{"ยอดคงค้าง / Balance due", amount(inv.BalanceDue())})
	}
	spec := docSpec{
		thTitle: "ใบแจ้งหนี้", enTitle: "INVOICE",
		meta:      meta,
		buyerName: inv.CustomerName, buyerAddress: inv.Address,
		buyerTaxID: inv.CustomerTaxID, buyerBranch: inv.CustomerTaxBranch,
		description: inv.ServiceName + siteLine(inv.SiteName), lineAmount: inv.Subtotal, qtyUnit: "งาน",
		totals: totals, words: inv.Total, currency: inv.Currency,
		imageKey:   inv.InvoiceNumber,
		copies:     []string{"ต้นฉบับ / Original", "สำเนา / Copy"},
		note:       fmt.Sprintf("เงื่อนไขการชำระเงิน: ภายใน %d วันนับจากวันที่ออกใบแจ้งหนี้ / Payment terms: %d days from the invoice date", dueDays, dueDays),
		signLabels: [2]string{"ผู้วางบิล / Issued by", "ผู้รับวางบิล / Received by"},
		signFor:    [2]string{"seller", "buyer"},
		signDate:   inv.IssuedAt,
		signDated:  [2]bool{true, false},
	}
	if inv.Status.Open() {
		spec.payAmount = inv.BalanceDue()
	}
	if depositPct > 0 && inv.Status == StatusIssued && inv.AmountPaid == 0 {
		due := inv.NetPayable()
		dep := math.Round(due*float64(depositPct)) / 100
		rest := due - dep
		spec.payAmount = dep
		spec.terms = fmt.Sprintf("ขอเรียกเก็บเงินมัดจำ %[1]d%% เพื่อยืนยันการจอง: %[2]s %[3]s (ยอดคงเหลือ %[2]s %[4]s)\n"+
			"Deposit due (%[1]d%%) to confirm the booking: %[2]s %[3]s (remaining balance %[2]s %[4]s)",
			depositPct, inv.Currency, amount(dep), amount(rest))
	}
	switch inv.Status {
	case StatusPaid:
		spec.stamp, spec.stampRGB = "ชำระแล้ว / PAID", [3]int{22, 163, 74}
	case StatusVoid:
		spec.stamp, spec.stampRGB = "ยกเลิก / VOID", [3]int{220, 38, 38}
	}
	return renderDoc(spec, company, pay)
}

// renderReceiptPDF produces the receipt for one payment. A VAT-registered
// company's receipt doubles as the tax invoice (ใบเสร็จรับเงิน/ใบกำกับภาษี):
// for services the tax point is when payment is received.
func renderReceiptPDF(rc Receipt, inv Invoice, company settings.Company, pay settings.PaymentSettings) ([]byte, error) {
	thTitle, enTitle := "ใบเสร็จรับเงิน", "RECEIPT"
	if rc.VATRegistered {
		thTitle, enTitle = "ใบเสร็จรับเงิน / ใบกำกับภาษี", "RECEIPT / TAX INVOICE"
	}
	meta := [][2]string{
		{"เลขที่ / No.", rc.ReceiptNumber},
		{"วันที่ / Date", thaiDate(rc.PaidAt)},
		{"ใบแจ้งหนี้ / Invoice", rc.InvoiceNumber},
		{"อ้างอิง / Booking", rc.BookingNumber},
	}
	desc := rc.ServiceName + "\nอ้างอิงใบแจ้งหนี้ / Invoice ref: " + rc.InvoiceNumber + siteLine(inv.SiteName)
	if rc.Gross() < inv.Total-0.005 {
		desc += "\n(ชำระบางส่วน / Partial payment)"
	}
	totals := [][2]string{{"รวมเงิน / Total amount", amount(rc.Subtotal)}}
	if rc.VAT > 0 {
		totals = append(totals, [2]string{fmt.Sprintf("ภาษีมูลค่าเพิ่ม / VAT %s%%", trimRate(rc.TaxRate)), amount(rc.VAT)})
	}
	totals = append(totals, [2]string{"จำนวนเงินรวมทั้งสิ้น / Grand total", amount(rc.Gross())})
	if rc.WHT > 0 {
		totals = append(totals, [2]string{fmt.Sprintf("ภาษีหัก ณ ที่จ่าย / WHT %s%%", trimRate(rc.WithholdingRate)), "-" + amount(rc.WHT)})
	}
	totals = append(totals, [2]string{"ยอดเงินสุทธิ / Total net received", amount(rc.Amount)})
	if bal := inv.BalanceDue(); bal > 0 && rc.Status == ReceiptValid {
		totals = append(totals, [2]string{"ยอดคงค้าง / Balance due", amount(bal)})
	}
	spec := docSpec{
		thTitle: thTitle, enTitle: enTitle,
		meta:      meta,
		buyerName: rc.CustomerName, buyerAddress: rc.Address,
		buyerTaxID: rc.CustomerTaxID, buyerBranch: rc.CustomerTaxBranch,
		description: desc, lineAmount: rc.Subtotal, qtyUnit: "งาน",
		totals: totals, words: rc.Gross(), currency: rc.Currency,
		imageKey: rc.ReceiptNumber,
		stamp:    "ได้รับเงินแล้ว / RECEIVED", stampRGB: [3]int{22, 163, 74},
		payMethods: receiptMethods(rc, pay),
		signLabels: [2]string{"ผู้มีอำนาจลงนาม / Authorized signature", "ผู้อนุมัติ/ผู้รับเงิน / Approved / Collected by"},
		signFor:    [2]string{"seller", ""},
		signDate:   rc.PaidAt,
		signDated:  [2]bool{true, false},
		copies:     []string{"ต้นฉบับ / Original", "สำเนา / Copy"},
	}
	if rc.Status == ReceiptCancelled {
		spec.stamp, spec.stampRGB = "ยกเลิก / CANCELLED", [3]int{220, 38, 38}
	}
	return renderDoc(spec, company, settings.PaymentSettings{})
}

// receiptMethods is the receipt's "paid by" checklist, the way Thai receipts
// print it: cash / transfer to our account / cheque / other, with the one
// used ticked and its reference filled in.
func receiptMethods(rc Receipt, pay settings.PaymentSettings) []checkItem {
	ref := strings.TrimSpace(rc.Reference)
	withRef := func(s string) string {
		if ref != "" {
			return s + " (" + ref + ")"
		}
		return s
	}
	bank := strings.Join(strings.Fields(pay.BankAccount), " ")
	transfer := "เงินโอนเข้าบัญชี / Transfer to account"
	if bank != "" {
		transfer += " " + bank
	}
	cheque := "เช็คเลขที่ / Cheque No. ____________"
	other := "อื่นๆ / Other ____________"
	switch rc.Method {
	case "cash":
	case "bank_transfer":
		transfer = withRef(transfer)
	case "promptpay":
		transfer = withRef("เงินโอน / Transfer: PromptPay")
	case "cheque":
		if ref != "" {
			cheque = "เช็คเลขที่ / Cheque No. " + ref
		}
	default:
		other = withRef("อื่นๆ / Other: " + methodLabel(rc.Method))
	}
	known := rc.Method == "cash" || rc.Method == "bank_transfer" || rc.Method == "promptpay" || rc.Method == "cheque"
	return []checkItem{
		{"เงินสด / Cash", rc.Method == "cash"},
		{transfer, rc.Method == "bank_transfer" || rc.Method == "promptpay"},
		{cheque, rc.Method == "cheque"},
		{other, !known},
	}
}

// checkbox draws a 3.2 mm box at (x, y), crossed when checked.
func checkbox(pdf *gofpdf.Fpdf, x, y float64, checked bool) {
	pdf.SetDrawColor(71, 85, 105)
	pdf.Rect(x, y, 3.2, 3.2, "D")
	if checked {
		pdf.SetLineWidth(0.4)
		pdf.Line(x+0.6, y+0.6, x+2.6, y+2.6)
		pdf.Line(x+2.6, y+0.6, x+0.6, y+2.6)
		pdf.SetLineWidth(0.2)
	}
	pdf.SetDrawColor(203, 213, 225)
}

func methodLabel(m string) string {
	switch m {
	case "cash":
		return "เงินสด / Cash"
	case "bank_transfer":
		return "โอนเงิน / Bank transfer"
	case "promptpay":
		return "PromptPay"
	case "credit_card":
		return "บัตรเครดิต / Card"
	case "line_pay":
		return "LINE Pay"
	case "cheque":
		return "เช็ค / Cheque"
	}
	return m
}

// docItem is one priced row of a document.
type docItem struct {
	desc      string
	qty, unit float64
	amount    float64
}

func newDocPDF() *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes(fontFamily, "", fontRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", fontBold)
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(false, 0)
	return pdf
}

// renderDoc prints the document once per copy (Thai practice: ต้นฉบับ for
// the customer, สำเนา for our files), each copy numbered "Page x/n".
func renderDoc(spec docSpec, company settings.Company, pay settings.PaymentSettings) ([]byte, error) {
	labels := spec.copies
	if len(labels) == 0 {
		labels = []string{spec.copyLabel}
	}
	// Every copy has the same pages: count them once.
	probe := newDocPDF()
	drawDoc(probe, spec, company, pay, labels[0])
	perCopy := max(1, probe.PageCount())

	pdf := newDocPDF()
	pdf.SetFooterFunc(func() {
		pdf.SetXY(15, 283)
		pdf.SetFont(fontFamily, "", 8)
		pdf.SetTextColor(148, 163, 184)
		pdf.CellFormat(180, 4, fmt.Sprintf("หน้า / Page %d/%d", (pdf.PageNo()-1)%perCopy+1, perCopy), "", 0, "R", false, 0, "")
	})
	for _, label := range labels {
		drawDoc(pdf, spec, company, pay, label)
	}
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("gofpdf: %w", err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gofpdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// drawDoc lays out one copy of the document, starting on a new page.
func drawDoc(pdf *gofpdf.Fpdf, spec docSpec, company settings.Company, pay settings.PaymentSettings, copyLabel string) {
	pdf.AddPage()

	ink := func(r, g, b int) { pdf.SetTextColor(r, g, b) }
	font := func(style string, size float64) { pdf.SetFont(fontFamily, style, size) }
	const left, right, width = 15.0, 195.0, 180.0

	// Seller block (left), beside the company logo.
	pdf.RegisterImageOptionsReader("logo", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(logoPNG))
	pdf.ImageOptions("logo", left, 13, 22, 22, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	const sellerX, sellerW = left + 25, 78.0
	sellerName := strings.TrimSpace(company.LegalName)
	if sellerName == "" {
		sellerName = company.Name
	}
	pdf.SetXY(sellerX, 15)
	font("B", 14)
	ink(30, 41, 59)
	pdf.MultiCell(sellerW, 6.5, sellerName, "", "L", false)
	// The English name under the Thai legal name, as on Thai letterheads.
	if en := strings.TrimSpace(company.Name); en != "" && en != sellerName {
		font("B", 11)
		pdf.SetX(sellerX)
		pdf.MultiCell(sellerW, 5.5, en, "", "L", false)
	}
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
	pdf.SetX(sellerX)
	pdf.MultiCell(sellerW, 4.6, strings.Join(seller, "\n"), "", "L", false)
	sellerBottom := max(pdf.GetY(), 36)

	// Title + meta (right).
	pdf.SetXY(115, 15)
	font("B", 15)
	ink(30, 58, 95)
	pdf.CellFormat(80, 7, spec.thTitle, "", 2, "R", false, 0, "")
	font("B", 10)
	pdf.CellFormat(80, 5, spec.enTitle, "", 2, "R", false, 0, "")
	if copyLabel != "" {
		font("", 9)
		ink(100, 116, 139)
		pdf.CellFormat(80, 5, copyLabel, "", 2, "R", false, 0, "")
		ink(30, 58, 95)
	}
	pdf.Ln(2)
	font("", 9.5)
	ink(30, 41, 59)
	for _, m := range spec.meta {
		pdf.SetX(123)
		pdf.CellFormat(29, 5, m[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(43, 5, m[1], "", 1, "R", false, 0, "")
	}
	y := pdf.GetY()
	if sellerBottom > y {
		y = sellerBottom
	}

	// Customer box.
	y += 5
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetFillColor(248, 250, 252)
	buyer := []string{spec.buyerName}
	if a := strings.TrimSpace(spec.buyerAddress); a != "" {
		buyer = append(buyer, a)
	}
	buyerTaxID := digitsOf(spec.buyerTaxID)
	font("", 10)
	lines := 0
	for _, l := range buyer {
		lines += max(1, int(math.Ceil(pdf.GetStringWidth(l)/(width-8))))
	}
	if buyerTaxID != "" {
		lines++
	}
	boxH := 9 + float64(lines)*5
	pdf.Rect(left, y, width, boxH, "FD")
	pdf.SetXY(left+3, y+2)
	font("B", 9.5)
	ink(100, 116, 139)
	pdf.CellFormat(0, 5, "ลูกค้า / Customer", "", 1, "L", false, 0, "")
	font("", 10)
	ink(30, 41, 59)
	pdf.SetX(left + 3)
	pdf.MultiCell(width-6, 5, strings.Join(buyer, "\n"), "", "L", false)
	// Tax ID with the Revenue Department's head office / branch boxes.
	if buyerTaxID != "" {
		ly := pdf.GetY()
		x := left + 3
		label := "เลขประจำตัวผู้เสียภาษี / Tax ID: " + formatTaxID(buyerTaxID)
		pdf.SetXY(x, ly)
		pdf.CellFormat(pdf.GetStringWidth(label)+6, 5, label, "", 0, "L", false, 0, "")
		x = pdf.GetX()
		branch := digitsOf(spec.buyerBranch)
		head := branch == "" || strings.Trim(branch, "0") == ""
		checkbox(pdf, x, ly+0.9, head)
		pdf.SetXY(x+4.5, ly)
		hl := "สำนักงานใหญ่ / Head office"
		pdf.CellFormat(pdf.GetStringWidth(hl)+6, 5, hl, "", 0, "L", false, 0, "")
		x = pdf.GetX()
		checkbox(pdf, x, ly+0.9, !head)
		pdf.SetXY(x+4.5, ly)
		bl := "สาขาที่ / Branch ________"
		if !head {
			bl = "สาขาที่ / Branch " + strings.Repeat("0", max(0, 5-len(branch))) + branch
		}
		pdf.CellFormat(50, 5, bl, "", 0, "L", false, 0, "")
	}
	y += boxH + 6

	// Line items. Long documents (many quote lines) continue on new pages
	// with the column header repeated.
	type col struct {
		label string
		w     float64
		align string
	}
	cols := []col{
		{"ลำดับ\nNo.", 14, "C"},
		{"รายละเอียด\nDescription", 117, "L"},
		{"จำนวน\nQty", 18, "C"},
		{"จำนวนเงิน\nAmount", 31, "R"},
	}
	if spec.unitPriceCol {
		cols = []col{
			{"ลำดับ\nNo.", 14, "C"},
			{"รายการ\nDescription", 86, "L"},
			{"จำนวน\nQty", 18, "C"},
			{"ราคาต่อหน่วย\nUnit price", 31, "R"},
			{"จำนวนเงิน\nAmount", 31, "R"},
		}
	}
	drawHead := func() {
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
	}
	newPage := func() {
		pdf.AddPage()
		y = 15
	}
	items := spec.items
	if len(items) == 0 {
		items = []docItem{{desc: spec.description, qty: 1, unit: spec.lineAmount, amount: spec.lineAmount}}
	}
	drawHead()
	for i, it := range items {
		font("", 10)
		// Estimate wrapped lines from the text width (SplitText miscounts
		// Thai, which has no spaces between words).
		lines := 0
		for _, para := range strings.Split(it.desc, "\n") {
			lines += max(1, int(math.Ceil(pdf.GetStringWidth(para)/(cols[1].w-3))))
		}
		h := float64(lines)*5.5 + 3
		if y+h > 262 {
			newPage()
			drawHead()
		}
		qty := qtyText(it.qty)
		if spec.qtyUnit != "" {
			qty += " " + spec.qtyUnit
		}
		row := []string{fmt.Sprint(i + 1), it.desc, qty, amount(it.amount)}
		if spec.unitPriceCol {
			row = []string{fmt.Sprint(i + 1), it.desc, qty, amount(it.unit), amount(it.amount)}
		}
		x := left
		for j, c := range cols {
			pdf.SetXY(x, y+1.5)
			pdf.MultiCell(c.w, 5.5, row[j], "", c.align, false)
			x += c.w
		}
		y += h
		if i < len(items)-1 {
			pdf.SetDrawColor(226, 232, 240)
			pdf.Line(left, y, right, y)
			pdf.SetDrawColor(203, 213, 225)
		}
	}
	y += 4
	pdf.Line(left, y, right, y)

	// Keep totals, payment, stamp and signatures together on the last page.
	need := float64(len(spec.totals))*7 + 9
	if strings.TrimSpace(pay.PromptPayID+pay.BankAccount) != "" && spec.payAmount > 0 {
		need += 43
	}
	if spec.stamp != "" {
		need += 16
	}
	if spec.terms != "" {
		need += 23
	}
	if n := len(spec.payMethods); n > 0 {
		need = max(need, float64(n)*5.5+30)
	}
	// The signature blocks start at 230 mm.
	if y+need > 229 {
		newPage()
	}

	// Totals (right) and amount in words (left).
	y += 3
	ty := y
	for i, t := range spec.totals {
		last := i == len(spec.totals)-1
		if last {
			font("B", 11)
			pdf.SetFillColor(241, 245, 249)
		} else {
			font("", 10)
		}
		pdf.SetXY(103, ty)
		pdf.CellFormat(62, 7, t[0], "", 0, "L", last, 0, "")
		pdf.CellFormat(30, 7, t[1], "", 1, "R", last, 0, "")
		ty += 7
	}
	pdf.SetXY(left, y)
	font("B", 9)
	ink(100, 116, 139)
	pdf.CellFormat(85, 5, "จำนวนเงินตัวอักษร / Amount in words", "", 2, "L", false, 0, "")
	font("B", 10.5)
	ink(30, 41, 59)
	pdf.MultiCell(85, 5.5, "("+thaibaht.Text(spec.words)+")", "", "L", false)
	if spec.currency != "" && spec.currency != "THB" {
		font("", 8.5)
		ink(100, 116, 139)
		pdf.SetX(left)
		pdf.MultiCell(85, 4.5, "Amounts in "+spec.currency, "", "L", false)
	}
	if spec.note != "" {
		pdf.Ln(2)
		font("", 9.5)
		ink(71, 85, 105)
		pdf.SetX(left)
		pdf.MultiCell(85, 5, spec.note, "", "L", false)
	}
	if len(spec.payMethods) > 0 {
		pdf.Ln(2)
		font("B", 9)
		ink(100, 116, 139)
		pdf.SetX(left)
		pdf.CellFormat(85, 5, "ชำระโดย / Paid by", "", 2, "L", false, 0, "")
		font("", 9.5)
		ink(30, 41, 59)
		for _, m := range spec.payMethods {
			my := pdf.GetY()
			checkbox(pdf, left, my+1, m.checked)
			pdf.SetXY(left+5, my)
			pdf.MultiCell(83, 5, m.label, "", "L", false)
			pdf.Ln(0.5)
		}
	}
	// Whichever column ends lower decides where the next block starts.
	y = max(ty, pdf.GetY()) + 6

	if spec.terms != "" {
		y = max(y, pdf.GetY()+4)
		font("B", 10.5)
		lines := 0
		for _, para := range strings.Split(spec.terms, "\n") {
			lines += max(1, int(math.Ceil(pdf.GetStringWidth(para)/(width-10))))
		}
		h := float64(lines)*5.6 + 6
		pdf.SetFillColor(255, 247, 237)
		pdf.SetDrawColor(251, 146, 60)
		pdf.Rect(left, y, width, h, "FD")
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetXY(left+5, y+3)
		ink(154, 52, 18)
		pdf.MultiCell(width-10, 5.6, spec.terms, "", "L", false)
		y += h + 6
	}

	// Payment: PromptPay QR for the amount due, plus bank details.
	payTarget := strings.TrimSpace(pay.PromptPayID)
	bank := strings.TrimSpace(pay.BankAccount)
	if (payTarget != "" || bank != "") && spec.payAmount > 0 {
		font("B", 10)
		ink(30, 41, 59)
		pdf.SetXY(left, y)
		pdf.CellFormat(0, 6, "ช่องทางการชำระเงิน / Payment", "", 1, "L", false, 0, "")
		textX := left
		if payTarget != "" && (spec.currency == "" || spec.currency == "THB") {
			if payload, err := promptpay.Payload(payTarget, spec.payAmount); err == nil {
				if png, err := promptpay.PNG(payload, 360); err == nil {
					name := "promptpay-" + spec.imageKey
					pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
					pdf.ImageOptions(name, left, y+7, 34, 34, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
					textX = left + 38
				}
			}
		}
		pdf.SetXY(textX, y+8)
		if payTarget != "" {
			d, _ := promptpay.Normalize(payTarget)
			font("", 10.5)
			ink(51, 65, 85)
			pdf.MultiCell(right-textX, 5.4, "สแกนเพื่อชำระด้วย PromptPay / Scan to pay with PromptPay\n"+
				"PromptPay: "+d+"\nยอดชำระ / Amount: "+spec.currency+" "+amount(spec.payAmount), "", "L", false)
			pdf.Ln(2)
		}
		if bank != "" {
			pdf.SetX(textX)
			font("B", 10.5)
			ink(51, 65, 85)
			pdf.CellFormat(right-textX, 5.6, "โอนเงินเข้าบัญชี / Bank transfer:", "", 2, "L", false, 0, "")
			pdf.SetX(textX)
			font("B", 12)
			ink(30, 41, 59)
			pdf.MultiCell(right-textX, 6.2, bank, "", "L", false)
		}
		y = max(y+43, pdf.GetY()+4)
	}

	if spec.stamp != "" {
		c := spec.stampRGB
		pdf.SetDrawColor(c[0], c[1], c[2])
		ink(c[0], c[1], c[2])
		font("B", 14)
		pdf.SetLineWidth(0.8)
		pdf.Rect(130, y, 65, 14, "D")
		pdf.SetXY(130, y+3.5)
		pdf.CellFormat(65, 7, spec.stamp, "", 0, "C", false, 0, "")
		pdf.SetLineWidth(0.2)
		pdf.SetDrawColor(203, 213, 225)
	}

	// Signatures, Thai style: "ในนาม <organisation>", the signing line,
	// the name in brackets, the role and the date.
	sy := 230.0
	for i, label := range spec.signLabels {
		if label == "" {
			continue
		}
		sx := left + float64(i)*95
		ink(71, 85, 105)
		var org string
		switch spec.signFor[i] {
		case "seller":
			org = sellerName
		case "buyer":
			org = spec.buyerName
		}
		if org != "" {
			font("", 8.5)
			pdf.SetXY(sx, sy)
			pdf.CellFormat(85, 4.5, "ในนาม / For "+org, "", 0, "C", false, 0, "")
		}
		font("", 9.5)
		pdf.Line(sx+8, sy+17, sx+77, sy+17)
		pdf.SetXY(sx, sy+18)
		pdf.CellFormat(85, 5, "(                                                  )", "", 2, "C", false, 0, "")
		pdf.SetX(sx)
		pdf.CellFormat(85, 5, label, "", 2, "C", false, 0, "")
		date := "วันที่ / Date ____/____/______"
		if spec.signDated[i] && !spec.signDate.IsZero() {
			date = "วันที่ / Date " + thaiDate(spec.signDate)
		}
		pdf.SetX(sx)
		pdf.CellFormat(85, 5, date, "", 0, "C", false, 0, "")
	}

	footer := strings.TrimSpace(company.InvoiceFooter)
	if footer == "" {
		footer = "ขอบคุณที่ใช้บริการ / Thank you for your business!"
	}
	pdf.SetXY(left, 271)
	font("", 9)
	ink(148, 163, 184)
	pdf.MultiCell(width, 4.5, footer, "", "C", false)
}

// qtyText prints a quantity without needless decimals (2, 1.5).
func qtyText(q float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", q), "0"), ".")
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
