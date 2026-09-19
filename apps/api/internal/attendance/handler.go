package attendance

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/attendance?cleaner=&from=&to=&limit=&offset=.
// from/to are work dates (YYYY-MM-DD); pagination.Parse only accepts RFC3339
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
	items, total, err := h.service.List(r.Context(), params.CleanerID, from, to, params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]RecordDTO, 0, len(items))
	for _, rec := range items {
		dtos = append(dtos, toDTO(rec))
	}
	response.JSON(w, http.StatusOK, pagination.Page[RecordDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// CheckIn handles POST /api/v1/attendance/check-in.
func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CheckActionRequest](w, r)
	if !ok {
		return
	}
	rec, err := h.service.CheckIn(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(rec))
}

// CheckOut handles POST /api/v1/attendance/check-out.
func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CheckActionRequest](w, r)
	if !ok {
		return
	}
	rec, err := h.service.CheckOut(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(rec))
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
