package invoices

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/odysight/crm/internal/settings"
)

// Quotations share the invoice layout, fonts and logo, so they are rendered
// here; the quotes package supplies the data. Staff download the PDF and send
// it to the customer themselves.

// QuoteDoc is everything printed on a quotation.
type QuoteDoc struct {
	Number     string
	Date       time.Time
	ValidUntil *time.Time
	Status     string // draft | sent | accepted | rejected | expired

	CustomerName string
	Address      string
	TaxID        string
	TaxBranch    string
	Phone        string
	SiteName     string

	Lines    []QuoteLine
	Subtotal float64
	TaxRate  float64
	Total    float64
	Currency string
	Notes    string
}

// QuoteLine is one priced row of a quotation.
type QuoteLine struct {
	Name        string
	Description string
	Quantity    float64
	UnitPrice   float64
	Amount      float64
}

// RenderQuotePDF produces a bilingual (Thai/English) quotation, ใบเสนอราคา.
func RenderQuotePDF(q QuoteDoc, company settings.Company) ([]byte, error) {
	meta := [][2]string{
		{"เลขที่ / No.", q.Number},
		{"วันที่ / Date", thaiDate(q.Date)},
	}
	if q.ValidUntil != nil {
		meta = append(meta, [2]string{"ยืนราคาถึง / Valid until", thaiDate(*q.ValidUntil)})
	}
	if s := strings.TrimSpace(q.SiteName); s != "" {
		meta = append(meta, [2]string{"สถานที่ / Site", s})
	}
	items := make([]docItem, 0, len(q.Lines))
	for _, l := range q.Lines {
		desc := strings.TrimSpace(l.Name)
		if d := strings.TrimSpace(l.Description); d != "" {
			desc += "\n" + d
		}
		items = append(items, docItem{desc: desc, qty: l.Quantity, unit: l.UnitPrice, amount: l.Amount})
	}
	totals := [][2]string{{"รวมเป็นเงิน / Subtotal", amount(q.Subtotal)}}
	if tax := q.Total - q.Subtotal; tax > 0.005 {
		totals = append(totals, [2]string{fmt.Sprintf("ภาษีมูลค่าเพิ่ม / VAT %s%%", trimRate(q.TaxRate)), amount(tax)})
	}
	totals = append(totals, [2]string{"จำนวนเงินรวมทั้งสิ้น / Grand total", amount(q.Total)})

	// The validity date is in the header (ยืนราคาถึง / Valid until).
	note := strings.TrimSpace(q.Notes)
	spec := docSpec{
		thTitle: "ใบเสนอราคา", enTitle: "QUOTATION",
		meta:      meta,
		buyerName: q.CustomerName, buyerAddress: q.Address,
		buyerTaxID: q.TaxID, buyerBranch: q.TaxBranch, buyerPhone: q.Phone,
		items: items, unitPriceCol: true,
		totals: totals, words: q.Total, currency: q.Currency,
		imageKey:   q.Number,
		note:       note,
		signLabels: [2]string{"ผู้อนุมัติสั่งซื้อ / Accepted by", "ผู้เสนอราคา / Quoted by"},
		signFor:    [2]string{"buyer", "seller"},
		signDate:   q.Date,
		signDated:  [2]bool{false, true},
		terms: fmt.Sprintf("เงื่อนไขการจอง: กรุณาชำระเงินมัดจำ %[1]d%% (%[2]s %[3]s) เพื่อยืนยันการจอง\n"+
			"Booking terms: a %[1]d%% deposit (%[2]s %[3]s) is required to confirm the booking.",
			DepositPercent, q.Currency, amount(depositOf(q.Total))),
	}
	switch q.Status {
	case "accepted":
		spec.stamp, spec.stampRGB = "อนุมัติแล้ว / ACCEPTED", [3]int{22, 163, 74}
	case "rejected":
		spec.stamp, spec.stampRGB = "ไม่อนุมัติ / DECLINED", [3]int{220, 38, 38}
	case "expired":
		spec.stamp, spec.stampRGB = "หมดอายุ / EXPIRED", [3]int{220, 38, 38}
	}
	return renderDoc(spec, company, settings.PaymentSettings{})
}

// QuotePDF renders a quotation with the company details from settings.
func (s *Service) QuotePDF(ctx context.Context, q QuoteDoc) ([]byte, error) {
	company, _ := s.billingSettings(ctx)
	return RenderQuotePDF(q, company)
}

// DepositPercent is the share of a quotation's total the customer pays to
// confirm a booking.
const DepositPercent = 50

func depositOf(total float64) float64 {
	return math.Round(total*DepositPercent) / 100
}
