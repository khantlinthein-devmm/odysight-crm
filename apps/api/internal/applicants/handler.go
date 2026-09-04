package applicants

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/applicants
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	applicants, err := h.service.List(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]ApplicantDTO, 0, len(applicants))
	for _, a := range applicants {
		dtos = append(dtos, toDTO(a))
	}
	response.JSON(w, http.StatusOK, dtos)
}

// Get handles GET /api/v1/applicants/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	applicant, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(applicant))
}

// Create handles POST /api/v1/applicants
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateApplicantRequest](w, r)
	if !ok {
		return
	}

	applicant, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(applicant))
}

// Update handles PATCH /api/v1/applicants/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateApplicantRequest](w, r)
	if !ok {
		return
	}

	applicant, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(applicant))
}

// Delete handles DELETE /api/v1/applicants/{id}
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
		response.Error(w, http.StatusBadRequest, "invalid applicant id")
		return 0, false
	}
	return id, true
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	return out, true
}
