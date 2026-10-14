package checklists

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	t, err := h.service.CreateTemplate(r.Context(), CallerFrom(r.Context()), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toTemplateDTO(t))
}

func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	req, ok := decodeJSON[UpdateTemplateRequest](w, r)
	if !ok {
		return
	}
	t, err := h.service.UpdateTemplate(r.Context(), CallerFrom(r.Context()), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toTemplateDTO(t))
}

// List handles GET /checklists (office only): ?state=pending|in_progress|
// submitted|approved|rework, cleanerId, from, to (YYYY-MM-DD), search.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 50, nil)
	q := r.URL.Query()
	f := ListFilter{State: strings.TrimSpace(q.Get("state")), Search: params.Search, Limit: params.Limit, Offset: params.Offset}
	if v := q.Get("cleanerId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id < 1 {
			response.Error(w, http.StatusBadRequest, "invalid cleanerId")
			return
		}
		f.CleanerID = id
	}
	for _, p := range []struct {
		key string
		dst **time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if v := q.Get(p.key); v != "" {
			d, err := time.ParseInLocation("2006-01-02", v, time.Local)
			if err != nil {
				response.Error(w, http.StatusBadRequest, p.key+" must be YYYY-MM-DD")
				return
			}
			*p.dst = &d
		}
	}
	items, total, err := h.service.List(r.Context(), CallerFrom(r.Context()), f)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]SummaryDTO, 0, len(items))
	for _, s := range items {
		dtos = append(dtos, toSummaryDTO(s))
	}
	response.JSON(w, http.StatusOK, pagination.Page[SummaryDTO]{Data: dtos, Total: total, Limit: f.Limit, Offset: f.Offset})
}

// Statuses handles GET /checklists/status?bookings=1,2,3 for job card badges.
func (h *Handler) Statuses(w http.ResponseWriter, r *http.Request) {
	var ids []int64
	for _, part := range strings.Split(r.URL.Query().Get("bookings"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id < 1 {
			response.Error(w, http.StatusBadRequest, "bookings must be a comma-separated list of ids")
			return
		}
		ids = append(ids, id)
	}
	if len(ids) > 200 {
		response.Error(w, http.StatusBadRequest, "at most 200 bookings per request")
		return
	}
	rows, err := h.service.Statuses(r.Context(), CallerFrom(r.Context()), ids)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	out := make([]StatusDTO, 0, len(rows))
	for _, s := range rows {
		out = append(out, StatusDTO{BookingID: s.BookingID, Status: string(s.Status), Submitted: s.Submitted,
			ReviewStatus: s.ReviewStatus, ReviewNote: s.ReviewNote, ItemsTotal: s.ItemsTotal, ItemsDone: s.ItemsDone})
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) GetByBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseBookingID(w, r)
	if !ok {
		return
	}
	c, err := h.service.GetByBooking(r.Context(), CallerFrom(r.Context()), bookingID)
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
	c, err := h.service.Create(r.Context(), CallerFrom(r.Context()), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c))
}

func (h *Handler) CompleteItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseID(w, r, "itemId", "invalid checklist item id")
	if !ok {
		return
	}
	req, ok := decodeJSON[CompleteItemRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.CompleteItem(r.Context(), CallerFrom(r.Context()), itemID, req)
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
	c, err := h.service.Confirm(r.Context(), CallerFrom(r.Context()), bookingID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c))
}

// Submit handles POST /checklists/bookings/{bookingId}/submit: the cleaner
// hands the finished checklist in, which locks it for office review.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseBookingID(w, r)
	if !ok {
		return
	}
	c, err := h.service.Submit(r.Context(), CallerFrom(r.Context()), bookingID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c))
}

// Review handles POST /checklists/bookings/{bookingId}/review with
// {"decision": "approved"|"rework", "note"}.
func (h *Handler) Review(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseBookingID(w, r)
	if !ok {
		return
	}
	req, ok := decodeJSON[ReviewRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.Review(r.Context(), CallerFrom(r.Context()), bookingID, req)
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
	itemID, ok := parseID(w, r, "itemId", "invalid checklist item id")
	if !ok {
		return
	}
	caller := CallerFrom(r.Context())
	if err := h.service.CheckItem(r.Context(), caller, itemID); err != nil {
		response.HandleError(w, r, err)
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
	c, err := h.service.AttachPhoto(r.Context(), caller, itemID, kind, url)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c))
}

// DownloadPhoto handles GET /checklists/photos/{file}. Staff-only: the route
// sits behind checklists.read so proof-of-work photos are never public, and
// cleaners only get photos of their own jobs.
func (h *Handler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "file")
	caller := CallerFrom(r.Context())
	if caller.Cleaner {
		itemID, ok := PhotoItemID(name)
		if !ok || h.service.CheckPhoto(r.Context(), caller, itemID) != nil {
			response.Error(w, http.StatusNotFound, "photo not found")
			return
		}
	}
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

func parseID(w http.ResponseWriter, r *http.Request, param, msg string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, msg)
		return 0, false
	}
	return id, true
}

func parseBookingID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return parseID(w, r, "bookingId", "invalid booking id")
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
