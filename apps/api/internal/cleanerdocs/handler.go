package cleanerdocs

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service  *Service
	maxBytes int64
}

func NewHandler(service *Service, maxUploadMB int) *Handler {
	if maxUploadMB <= 0 {
		maxUploadMB = 8
	}
	return &Handler{service: service, maxBytes: int64(maxUploadMB) << 20}
}

// List handles GET /cleaner-documents?cleanerId= (numbers masked).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	cleanerID, ok := parseQueryID(w, r, "cleanerId")
	if !ok {
		return
	}
	docs, err := h.service.List(r.Context(), cleanerID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, docs)
}

// Statuses handles GET /cleaner-documents/status?cleanerId= (type + expiry only).
func (h *Handler) Statuses(w http.ResponseWriter, r *http.Request) {
	cleanerID, ok := parseQueryID(w, r, "cleanerId")
	if !ok {
		return
	}
	docs, err := h.service.Statuses(r.Context(), cleanerID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, docs)
}

// Expiring handles GET /cleaner-documents/expiring?days=60.
func (h *Handler) Expiring(w http.ResponseWriter, r *http.Request) {
	days := ExpiringWindowDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "days must be a number")
			return
		}
		days = n
	}
	items, err := h.service.Expiring(r.Context(), days)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// Get handles GET /cleaner-documents/{docId}: full number, audited.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseDocID(w, r)
	if !ok {
		return
	}
	doc, err := h.service.Reveal(r.Context(), id, userID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.JSON(w, http.StatusOK, doc)
}

// Create handles POST /cleaner-documents (multipart: cleanerId, type, number,
// issueDate, expiryDate, notes, optional file).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Cap the body and keep the whole form in memory so an unencrypted copy of
	// the scan never spills into a temp file on disk.
	limit := h.maxBytes + 1<<20
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := r.ParseMultipartForm(limit); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.Error(w, http.StatusRequestEntityTooLarge, "file exceeds the upload limit")
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid multipart body")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	cleanerID, err := strconv.ParseInt(r.FormValue("cleanerId"), 10, 64)
	if err != nil || cleanerID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid cleanerId")
		return
	}
	in := FieldsInput{
		Type:       formPtr(r, "type"),
		Number:     formPtr(r, "number"),
		IssueDate:  formPtr(r, "issueDate"),
		ExpiryDate: formPtr(r, "expiryDate"),
		Notes:      formPtr(r, "notes"),
	}

	var upload *Upload
	if file, header, err := r.FormFile("file"); err == nil {
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(io.LimitReader(file, h.maxBytes+1))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "could not read file")
			return
		}
		if int64(len(data)) > h.maxBytes {
			response.Error(w, http.StatusRequestEntityTooLarge, "file exceeds the upload limit")
			return
		}
		upload = &Upload{Data: data, OriginalName: header.Filename}
	} else if !errors.Is(err, http.ErrMissingFile) {
		response.Error(w, http.StatusBadRequest, "invalid file")
		return
	}

	doc, err := h.service.Create(r.Context(), cleanerID, userID(r), in, upload)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, doc)
}

// Update handles PATCH /cleaner-documents/{docId} (JSON metadata only).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseDocID(w, r)
	if !ok {
		return
	}
	var in FieldsInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	doc, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, doc)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseDocID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// File handles GET /cleaner-documents/{docId}/file: decrypted in memory,
// never cached, audited.
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	id, ok := parseDocID(w, r)
	if !ok {
		return
	}
	doc, data, err := h.service.File(r.Context(), id, userID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	ext := allowedTypes[doc.ContentType]
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Disposition", `inline; filename="`+string(doc.Type)+"-"+strconv.FormatInt(doc.ID, 10)+ext+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

func userID(r *http.Request) int64 {
	id, _ := auth.IdentityFromContext(r.Context())
	return id.UserID
}

func formPtr(r *http.Request, key string) *string {
	if _, ok := r.MultipartForm.Value[key]; !ok {
		return nil
	}
	v := r.FormValue(key)
	return &v
}

func parseDocID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "docId"), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid document id")
		return 0, false
	}
	return id, true
}

func parseQueryID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid "+key)
		return 0, false
	}
	return id, true
}
