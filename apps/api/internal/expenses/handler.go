package expenses

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/expenses?from=&to=&category=&search=&limit=&offset=.
// from/to are spend dates (YYYY-MM-DD); pagination.Parse only accepts RFC3339
// timestamps, so they are read off the query string here instead.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 50, nil)
	from, ok := parseDateParam(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseDateParam(w, r, "to")
	if !ok {
		return
	}
	filters := Filters{
		From:     from,
		To:       to,
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
	}
	items, total, err := h.service.List(r.Context(), filters, params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]ExpenseDTO, 0, len(items))
	for _, e := range items {
		dtos = append(dtos, toDTO(e))
	}
	response.JSON(w, http.StatusOK, pagination.Page[ExpenseDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateExpenseRequest](w, r)
	if !ok {
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	e, err := h.service.Create(r.Context(), req, identity.UserID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(e))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	req, ok := decodeJSON[UpdateExpenseRequest](w, r)
	if !ok {
		return
	}
	e, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(e))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid expense id")
		return 0, false
	}
	return id, true
}

func parseDateParam(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return "", true
	}
	if _, err := time.Parse(dateFormat, raw); err != nil {
		response.Error(w, http.StatusBadRequest, key+" must be a YYYY-MM-DD date")
		return "", false
	}
	return raw, true
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
