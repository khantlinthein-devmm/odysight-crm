package invoices

import (
	"context"
	"fmt"
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
		{"วันที่ / Date", q.Date.Format("02/01/2006")},
	}
	if q.ValidUntil != nil {
		meta = append(meta, [2]string{"ยืนราคาถึง / Valid until", q.ValidUntil.Format("02/01/2006")})
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

	note := strings.TrimSpace(q.Notes)
	if q.ValidUntil != nil {
		v := "ราคานี้ยืนถึงวันที่ " + q.ValidUntil.Format("02/01/2006") + " / Prices valid until " + q.ValidUntil.Format("2 Jan 2006")
		if note != "" {
			note += "\n\n" + v
		} else {
			note = v
		}
	}
	spec := docSpec{
		thTitle: "ใบเสนอราคา", enTitle: "QUOTATION",
		meta:      meta,
		buyerName: q.CustomerName, buyerAddress: q.Address,
		buyerTaxID: q.TaxID, buyerBranch: q.TaxBranch,
		items:  items,
		totals: totals, words: q.Total, currency: q.Currency,
		imageKey:   q.Number,
		note:       note,
		signLabels: [2]string{"ผู้เสนอราคา / Quoted by", "ผู้อนุมัติ / Accepted by"},
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
