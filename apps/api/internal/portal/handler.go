package portal

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service      *Service
	secureCookie bool
	sessionTTL   time.Duration
}

func NewHandler(service *Service, secureCookie bool, sessionTTL time.Duration) *Handler {
	if sessionTTL <= 0 {
		sessionTTL = tokenTTLDefault
	}
	return &Handler{service: service, secureCookie: secureCookie, sessionTTL: sessionTTL}
}

func (h *Handler) setPortalCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearPortalCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// Login handles POST /api/v1/portal/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[LoginRequest](w, r)
	if !ok {
		return
	}
	token, customer, err := h.service.Login(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	h.setPortalCookie(w, token)
	response.JSON(w, http.StatusOK, Token{Token: token, Customer: customer})
}

// Logout handles POST /api/v1/portal/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	h.clearPortalCookie(w)
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Me handles GET /api/v1/portal/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.Me(r.Context(), CustomerID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

// ChangePassword handles PATCH /api/v1/portal/me/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[changePasswordRequest](w, r)
	if !ok {
		return
	}
	if err := h.service.ChangePassword(r.Context(), CustomerID(r), req.CurrentPassword, req.NewPassword); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Bookings handles GET /api/v1/portal/bookings
func (h *Handler) Bookings(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Bookings(r.Context(), CustomerID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	if items == nil {
		items = []Booking{}
	}
	response.JSON(w, http.StatusOK, items)
}

// Sites handles GET /api/v1/portal/sites
func (h *Handler) Sites(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Sites(r.Context(), CustomerID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// Services handles GET /api/v1/portal/services
func (h *Handler) Services(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Services(r.Context(), CustomerID(r))
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// CreateBooking handles POST /api/v1/portal/bookings
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateBookingRequest](w, r)
	if !ok {
		return
	}
	b, err := h.service.CreateBooking(r.Context(), CustomerID(r), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, b)
}

// CancelBooking handles POST /api/v1/portal/bookings/{id}/cancel
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "id")
	bookingID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || bookingID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid booking id")
		return
	}
	if err := h.service.CancelBooking(r.Context(), CustomerID(r), bookingID); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Feedback handles POST /api/v1/portal/bookings/{id}/feedback
func (h *Handler) Feedback(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "id")
	bookingID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || bookingID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid booking id")
		return
	}
	req, ok := decodeJSON[feedbackRequest](w, r)
	if !ok {
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		response.Error(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}
	if err := h.service.SubmitFeedback(r.Context(), CustomerID(r), bookingID, req.Rating, req.Comment); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type feedbackRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
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