package chat

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func identity(w http.ResponseWriter, r *http.Request) (auth.Identity, bool) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
	}
	return id, ok
}

func idParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || v < 1 {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return v, true
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// Contacts handles GET /chat/contacts
func (h *Handler) Contacts(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	out, err := h.service.Contacts(r.Context(), me)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": out})
}

// Conversations handles GET /chat/conversations
func (h *Handler) Conversations(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	out, err := h.service.Conversations(r.Context(), me)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": out})
}

// Unread handles GET /chat/unread
func (h *Handler) Unread(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	n, err := h.service.UnreadTotal(r.Context(), me)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]int{"unread": n})
}

// Open handles POST /chat/conversations {"userId": n}
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	var req struct {
		UserID int64 `json:"userId"`
	}
	if !decode(w, r, &req) {
		return
	}
	c, err := h.service.Open(r.Context(), me, req.UserID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

// Messages handles GET /chat/conversations/{id}/messages?after=&before=&limit=
func (h *Handler) Messages(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	q := r.URL.Query()
	after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	out, err := h.service.Messages(r.Context(), me, id, after, before, limit)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": out})
}

// Send handles POST /chat/conversations/{id}/messages — JSON {"body"} for
// text, multipart (file, kind, caption, durationMs) for photo and voice.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var msg Message
	var err error
	if ct := r.Header.Get("Content-Type"); len(ct) >= 19 && ct[:19] == "multipart/form-data" {
		r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid upload (max 8 MB)")
			return
		}
		defer func() { _ = r.MultipartForm.RemoveAll() }()
		file, _, ferr := r.FormFile("file")
		if ferr != nil {
			response.Error(w, http.StatusBadRequest, "file is required")
			return
		}
		defer file.Close()
		duration, _ := strconv.Atoi(r.FormValue("durationMs"))
		msg, err = h.service.SendFile(r.Context(), me, id, r.FormValue("kind"), file, r.FormValue("caption"), duration)
	} else {
		var req struct {
			Body string `json:"body"`
		}
		if !decode(w, r, &req) {
			return
		}
		msg, err = h.service.SendText(r.Context(), me, id, req.Body)
	}
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, msg)
}

// Read handles POST /chat/conversations/{id}/read {"messageId": n}
func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		MessageID int64 `json:"messageId"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.MarkRead(r.Context(), me, id, req.MessageID); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// File handles GET /chat/files/{name}
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	path, ct, err := h.service.File(r.Context(), me, chi.URLParam(r, "name"))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		response.Error(w, http.StatusNotFound, "file not found")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		response.Error(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	// ServeContent handles Range requests, which audio players rely on.
	http.ServeContent(w, r, "", st.ModTime(), f)
}

// Groups handles GET /chat/groups
func (h *Handler) Groups(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Groups(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": out})
}

// CreateGroup handles POST /chat/groups
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	h.saveGroup(w, r, 0)
}

// UpdateGroup handles PUT-like PATCH /chat/groups/{id}
func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	h.saveGroup(w, r, id)
}

func (h *Handler) saveGroup(w http.ResponseWriter, r *http.Request, id int64) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	var req SaveGroupRequest
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.SaveGroup(r.Context(), me, id, req); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteGroup handles DELETE /chat/groups/{id}
func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteGroup(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PushKey handles GET /chat/push/key
func (h *Handler) PushKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.service.PushPublicKey(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"publicKey": key})
}

// Subscribe handles POST /chat/push/subscribe (a PushSubscription JSON).
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	var req SubscribeRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	if err := dec.Decode(&req); err != nil { // browsers add expirationTime etc.
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.Subscribe(r.Context(), me, req); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Unsubscribe handles POST /chat/push/unsubscribe {"endpoint"}
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	me, ok := identity(w, r)
	if !ok {
		return
	}
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.Unsubscribe(r.Context(), me, req.Endpoint); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
