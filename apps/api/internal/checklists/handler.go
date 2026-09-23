package checklists

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
	photos  *PhotoStore
}

func NewHandler(service *Service, photos *PhotoStore) *Handler {
	if photos == nil {
		photos = NewPhotoStore("uploads", 8)
	}
	return &Handler{service: service, photos: photos}
}

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.ListTemplates(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]TemplateDTO, 0, len(items))
	for _, t := range items {
		dtos = append(dtos, toTemplateDTO(t))
	}
	response.JSON(w, http.StatusOK, pagination.Page[TemplateDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateTemplateRequest](w, r)
	if !ok {
		return
	}
	t, err := h.service.CreateTemplate(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toTemplateDTO(t))
}

func (h *Handler) GetByBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseBookingID(w, r)
	if !ok {
		return
	}
	c, err := h.service.GetByBooking(r.Context(), bookingID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateChecklistRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c))
}

func (h *Handler) CompleteItem(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "itemId")
	itemID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || itemID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid checklist item id")
		return
	}
	req, ok := decodeJSON[CompleteItemRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.CompleteItem(r.Context(), itemID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c))
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseBookingID(w, r)
	if !ok {
		return
	}
	req, ok := decodeJSON[ConfirmChecklistRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.Confirm(r.Context(), bookingID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c))
}

// UploadPhoto handles POST /checklists/items/{itemId}/photo (multipart:
// kind=before|after, file=image). Files land in private local storage; the
// recorded URL is only downloadable through the authenticated GET endpoint.
func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "itemId")
	itemID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || itemID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid checklist item id")
		return
	}
	if err := r.ParseMultipartForm(h.photos.maxBytes); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart body")
		return
	}
	kind := r.FormValue("kind")
	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() { _ = file.Close() }()
	url, err := h.photos.Save(itemID, kind, file, header)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := h.service.AttachPhoto(r.Context(), itemID, kind, url)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c))
}

// DownloadPhoto handles GET /checklists/photos/{file}. Staff-only: the route
// sits behind checklists.read so proof-of-work photos are never public.
func (h *Handler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "file")
	f, contentType, err := h.photos.Open(name)
	if err != nil {
		response.Error(w, http.StatusNotFound, "photo not found")
		return
	}
	defer func() { _ = f.Close() }()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if _, err := io.Copy(w, f); err != nil {
		return
	}
}

func parseBookingID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "bookingId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid booking id")
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
