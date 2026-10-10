package supplies

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermSuppliesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermSuppliesManage)).Post("/", h.Create)
	r.With(az.Require(auth.PermSuppliesRead)).Get("/movements", h.Movements)
	r.With(az.Require(auth.PermSuppliesRead)).Get("/site-costs", h.SiteCosts)
	r.Route("/{id}", func(r chi.Router) {
		r.With(az.Require(auth.PermSuppliesManage)).Patch("/", h.Update)
		r.With(az.Require(auth.PermSuppliesManage)).Post("/movements", h.Record)
	})
	return r
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	return out, true
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid supply id")
		return 0, false
	}
	return id, true
}

// List handles GET /api/v1/supplies?all=true (all includes inactive items).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("all") == "true")
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[CreateRequest](w, r)
	if !ok {
		return
	}
	s, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, s)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	req, ok := decode[UpdateRequest](w, r)
	if !ok {
		return
	}
	s, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, s)
}

// Record handles POST /api/v1/supplies/{id}/movements.
func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	req, ok := decode[MovementRequest](w, r)
	if !ok {
		return
	}
	ident, _ := auth.IdentityFromContext(r.Context())
	m, s, err := h.service.Record(r.Context(), id, ident.UserID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"movement": m, "supply": s})
}

// Movements handles GET /api/v1/supplies/movements?from=&to=&siteId=&kind=.
func (h *Handler) Movements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := Range(q.Get("from"), q.Get("to"), time.Now())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	siteID, _ := strconv.ParseInt(q.Get("siteId"), 10, 64)
	items, err := h.service.Movements(r.Context(), MovementFilters{From: from, To: to, SiteID: siteID, Kind: q.Get("kind")})
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// SiteCosts handles GET /api/v1/supplies/site-costs?from=&to=.
func (h *Handler) SiteCosts(w http.ResponseWriter, r *http.Request) {
	from, to, err := Range(r.URL.Query().Get("from"), r.URL.Query().Get("to"), time.Now())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	items, err := h.service.SiteCosts(r.Context(), from, to)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}
