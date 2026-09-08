package invoices

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf/v2"

	"github.com/odysight/crm/internal/settings"
)

// renderInvoicePDF produces a single-page A4 invoice, branded with the
// company's name, address and phone from settings.
// Uses the built-in Helvetica core font (WinAnsi), so ASCII-only text renders best.
func renderInvoicePDF(inv Invoice, company settings.Company) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	ink := func(r, g, b int) {
		pdf.SetTextColor(r, g, b)
	}

	// Header: company block on the left, title + meta on the right.
	if strings.TrimSpace(company.Name) != "" {
		pdf.SetFont("Helvetica", "B", 13)
		ink(30, 41, 59)
		pdf.MultiCell(90, 6, company.Name, "", "L", false)

		var lines []string
		if strings.TrimSpace(company.Address) != "" {
			lines = append(lines, company.Address)
		}
		if strings.TrimSpace(company.Phone) != "" {
			lines = append(lines, company.Phone)
		}
		if len(lines) > 0 {
			pdf.SetFont("Helvetica", "", 9)
			ink(100, 116, 139)
			pdf.MultiCell(90, 4.5, strings.Join(lines, "\n"), "", "L", false)
		}
	}

	pdf.SetXY(115, 20)
	pdf.SetFont("Helvetica", "B", 16)
	ink(30, 41, 59)
	pdf.Cell(0, 8, "INVOICE")
	pdf.SetXY(115, 30)
	pdf.SetFont("Helvetica", "", 10)
	ink(30, 41, 59)
	pdf.MultiCell(75, 5,
		"Invoice: "+inv.InvoiceNumber+"\n"+
			"Booking: "+inv.BookingNumber+"\n"+
			"Issued: "+inv.IssuedAt.Format("02/01/2006")+"\n"+
			"Status: "+string(inv.Status),
		"", "R", false)

	y := pdf.GetY()
	if y < 52 {
		y = 52
	}
	pdf.SetY(y)

	// Bill to.
	pdf.SetFont("Helvetica", "B", 10)
	ink(30, 41, 59)
	pdf.Cell(0, 6, "BILL TO")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 6, inv.CustomerName+"\n"+inv.Address, "", "L", false)
	pdf.Ln(10)

	// Line item.
	pdf.SetFont("Helvetica", "B", 10)
	ink(30, 41, 59)
	pdf.SetFillColor(241, 245, 249)
	pdf.Cell(90, 8, "Description")
	pdf.Cell(30, 8, "Qty")
	pdf.Cell(0, 8, "Amount")
	pdf.Ln(8)
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(90, 6, inv.ServiceName, "", "L", false)
	pdf.Cell(30, 6, "1")
	pdf.Cell(0, 6, money(inv.Currency, inv.Subtotal))
	pdf.Ln(16)

	// Totals.
	pdf.SetFont("Helvetica", "", 10)
	ink(100, 116, 139)
	pdf.Cell(80, 7, "Subtotal")
	pdf.Cell(0, 7, money(inv.Currency, inv.Subtotal))
	pdf.Ln(8)
	if inv.TaxRate > 0 {
		pdf.Cell(80, 7, fmt.Sprintf("Tax (%.2f%%)", inv.TaxRate))
		pdf.Cell(0, 7, money(inv.Currency, inv.TaxAmount))
		pdf.Ln(8)
	}
	pdf.SetFont("Helvetica", "B", 12)
	ink(30, 41, 59)
	pdf.Cell(80, 8, "Total")
	pdf.Cell(0, 8, money(inv.Currency, inv.Total))
	pdf.Ln(18)

	// Footer pinned near the bottom of the page (company note or a default).
	footer := strings.TrimSpace(company.InvoiceFooter)
	if footer == "" {
		footer = "Thank you for your business!"
	}
	if y := pdf.GetY(); y < 270 {
		pdf.SetY(270)
	}
	pdf.SetFont("Helvetica", "", 9)
	ink(148, 163, 184)
	pdf.MultiCell(0, 5, footer, "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gofpdf output: %w", err)
	}
	return buf.Bytes(), nil
}

func money(currency string, amount float64) string {
	return fmt.Sprintf("%s %0.2f", currency, amount)
}