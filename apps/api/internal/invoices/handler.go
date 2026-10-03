package invoices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/invoices
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]InvoiceDTO, 0, len(items))
	for _, inv := range items {
		dtos = append(dtos, toDTO(inv))
	}
	response.JSON(w, http.StatusOK, pagination.Page[InvoiceDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// Get handles GET /api/v1/invoices/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	inv, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(inv))
}

// Create handles POST /api/v1/invoices
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateInvoiceRequest](w, r)
	if !ok {
		return
	}

	inv, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(inv))
}

// Update handles PATCH /api/v1/invoices/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateInvoiceRequest](w, r)
	if !ok {
		return
	}

	inv, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(inv))
}

// Email handles POST /api/v1/invoices/{id}/email
func (h *Handler) Email(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.SendEmail(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// MarkUnpaid handles POST /api/v1/invoices/{id}/mark-unpaid
func (h *Handler) MarkUnpaid(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	inv, err := h.service.MarkUnpaid(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(inv))
}

// PDF handles GET /api/v1/invoices/{id}/pdf
func (h *Handler) PDF(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	// ?deposit=50 renders a deposit request (the only supported share).
	depositPct := 0
	if v := r.URL.Query().Get("deposit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n != DepositPercent {
			response.Error(w, http.StatusBadRequest, fmt.Sprintf("deposit must be %d", DepositPercent))
			return
		}
		depositPct = n
	}

	inv, pdfBytes, err := h.service.PDF(r.Context(), id, depositPct)
	if err != nil {
		response.HandleError(w, r, fmt.Errorf("render invoice %d: %w", id, err))
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition",
		`inline; filename="`+inv.InvoiceNumber+map[bool]string{true: "-deposit", false: ""}[depositPct > 0]+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// PromptPay handles GET /api/v1/invoices/{id}/promptpay.png — the payment QR
// for the invoice's net payable amount.
func (h *Handler) PromptPay(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	img, err := h.service.PromptPayQR(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img)
}

// RecordPayment handles POST /api/v1/invoices/{id}/payments
func (h *Handler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	req, ok := decodeJSON[RecordPaymentRequest](w, r)
	if !ok {
		return
	}
	inv, rc, err := h.service.RecordPayment(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, PaymentResultDTO{Invoice: toDTO(inv), Receipt: toReceiptDTO(rc)})
}

// Collect handles POST /api/v1/invoices/collect — bill a booking
// and record its payment in one step.
func (h *Handler) Collect(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CollectRequest](w, r)
	if !ok {
		return
	}
	inv, rc, err := h.service.Collect(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, PaymentResultDTO{Invoice: toDTO(inv), Receipt: toReceiptDTO(rc)})
}

// InvoiceReceipts handles GET /api/v1/invoices/{id}/receipts
func (h *Handler) InvoiceReceipts(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	h.listReceipts(w, r, id)
}

// ListReceipts handles GET /api/v1/receipts
func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	h.listReceipts(w, r, 0)
}

func (h *Handler) listReceipts(w http.ResponseWriter, r *http.Request, invoiceID int64) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.ListReceipts(r.Context(), params, invoiceID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]ReceiptDTO, 0, len(items))
	for _, rc := range items {
		dtos = append(dtos, toReceiptDTO(rc))
	}
	response.JSON(w, http.StatusOK, pagination.Page[ReceiptDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// GetReceipt handles GET /api/v1/receipts/{id}
func (h *Handler) GetReceipt(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	rc, err := h.service.GetReceipt(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toReceiptDTO(rc))
}

// ReceiptPDF handles GET /api/v1/receipts/{id}/pdf
func (h *Handler) ReceiptPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	rc, pdfBytes, err := h.service.ReceiptPDF(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+rc.ReceiptNumber+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// EmailReceipt handles POST /api/v1/receipts/{id}/email
func (h *Handler) EmailReceipt(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.service.SendReceipt(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	if dec.More() {
		response.Error(w, http.StatusBadRequest, "invalid JSON body: trailing data")
		return out, false
	}
	return out, true
}
