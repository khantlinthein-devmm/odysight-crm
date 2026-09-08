package auth

import (
	"net/http"
	"time"

	"github.com/odysight/crm/pkg/response"
)

const sessionCookieName = "odysight_session"

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

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
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

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
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

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[LoginRequest](w, r)
	if !ok {
		return
	}

	token, user, err := h.service.Login(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	h.setSessionCookie(w, token)
	response.JSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	h.clearSessionCookie(w)
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	user, err := h.service.Me(r.Context(), id.UserID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	req, ok := decodeJSON[ChangePasswordRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(); err != nil {
		response.HandleError(w, r, err)
		return
	}
	if err := h.service.ChangePassword(r.Context(), id.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
