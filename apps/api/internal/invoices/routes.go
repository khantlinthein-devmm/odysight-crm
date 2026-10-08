package invoices

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermInvoicesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermInvoicesCreate)).Post("/", h.Create)
	r.With(az.Require(auth.PermInvoicesCreate), az.Require(auth.PermPaymentsCreate)).Post("/collect", h.Collect)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermInvoicesRead))
		r.Get("/", h.Get)
		r.Get("/pdf", h.PDF)
		r.Get("/promptpay.png", h.PromptPay)
		r.Get("/receipts", h.InvoiceReceipts)
		r.With(az.Require(auth.PermPaymentsRead)).Get("/receipt-summary/pdf", h.SummaryReceiptPDF)
		r.With(az.Require(auth.PermPaymentsCreate)).Post("/payments", h.RecordPayment)
		r.With(az.Require(auth.PermInvoicesUpdate)).Post("/email", h.Email)
		r.With(az.Require(auth.PermInvoicesUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermPaymentsUpdate)).Post("/mark-unpaid", h.MarkUnpaid)
	})
	return r
}

// ReceiptRoutes serves /receipts. Receipts are payment records, so they
// follow the payments permissions.
func ReceiptRoutes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.Use(az.Require(auth.PermPaymentsRead))
	r.Get("/", h.ListReceipts)
	r.Get("/{id}", h.GetReceipt)
	r.Get("/{id}/pdf", h.ReceiptPDF)
	r.With(az.Require(auth.PermPaymentsUpdate)).Post("/{id}/email", h.EmailReceipt)
	return r
}
