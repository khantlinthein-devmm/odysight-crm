package complaints

import (
	"encoding/json"
	"net/http"
	"strconv"
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

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermComplaintsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermComplaintsManage)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.With(az.Require(auth.PermComplaintsRead)).Get("/", h.Get)
		r.With(az.Require(auth.PermComplaintsManage)).Patch("/", h.Update)
		r.With(az.Require(auth.PermComplaintsManage), az.Require(auth.PermBookingsCreate)).Post("/reclean", h.Reclean)
	})
	return r
}

type DTO struct {
	ID               int64      `json:"id"`
	Number           string     `json:"complaintNumber"`
	CustomerID       int64      `json:"customerId"`
	CustomerName     string     `json:"customerName"`
	SiteID           *int64     `json:"siteId"`
	SiteName         string     `json:"siteName"`
	BookingID        *int64     `json:"bookingId"`
	BookingNumber    string     `json:"bookingNumber"`
	Category         string     `json:"category"`
	Severity         string     `json:"severity"`
	Channel          string     `json:"channel"`
	Description      string     `json:"description"`
	Status           Status     `json:"status"`
	DueAt            time.Time  `json:"dueAt"`
	Overdue          bool       `json:"overdue"`
	Resolution       string     `json:"resolution"`
	ResolvedAt       *time.Time `json:"resolvedAt"`
	RecleanBookingID *int64     `json:"recleanBookingId"`
	RecleanNumber    string     `json:"recleanBookingNumber"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func toDTO(c Complaint, now time.Time) DTO {
	return DTO{c.ID, c.Number, c.CustomerID, c.CustomerName, c.SiteID, c.SiteName, c.BookingID, c.BookingNumber,
		c.Category, c.Severity, c.Channel, c.Description, c.Status, c.DueAt, c.Overdue(now), c.Resolution,
		c.ResolvedAt, c.RecleanBookingID, c.RecleanNumber, c.CreatedAt, c.UpdatedAt}
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
		response.Error(w, http.StatusBadRequest, "invalid complaint id")
		return 0, false
	}
	return id, true
}

func userID(r *http.Request) int64 {
	id, _ := auth.IdentityFromContext(r.Context())
	return id.UserID
}

// List handles GET /api/v1/complaints?status=&customerId=&overdue=true&search=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 50, nil)
	q := r.URL.Query()
	customerID, _ := strconv.ParseInt(q.Get("customerId"), 10, 64)
	items, total, err := h.service.List(r.Context(), Filters{
		Status: q.Get("status"), CustomerID: customerID, Overdue: q.Get("overdue") == "true",
		Search: params.Search, Limit: params.Limit, Offset: params.Offset,
	})
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	now := time.Now()
	out := make([]DTO, 0, len(items))
	for _, c := range items {
		out = append(out, toDTO(c, now))
	}
	response.JSON(w, http.StatusOK, pagination.Page[DTO]{Data: out, Total: total, Limit: params.Limit, Offset: params.Offset})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	c, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c, time.Now()))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[CreateRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.Create(r.Context(), req, userID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c, time.Now()))
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
	c, err := h.service.Update(r.Context(), id, req, userID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(c, time.Now()))
}

// Reclean handles POST /api/v1/complaints/{id}/reclean.
func (h *Handler) Reclean(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	req, ok := decode[RecleanRequest](w, r)
	if !ok {
		return
	}
	c, err := h.service.Reclean(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(c, time.Now()))
}
