package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

const (
	sessionCookieName = "odysight_session"
	// deviceCookieName marks a device the user chose to remember for 30 days.
	deviceCookieName = "odysight_device"
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

	device := ""
	if c, err := r.Cookie(deviceCookieName); err == nil {
		device = c.Value
	}
	res, err := h.service.Login(r.Context(), req, device)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	if res.MFARequired {
		// No session yet: the code step (POST /auth/login/2fa) issues it.
		response.JSON(w, http.StatusOK, LoginResponse{User: res.User, MFARequired: true, MFAToken: res.MFAToken})
		return
	}
	h.setSessionCookie(w, res.Token)
	response.JSON(w, http.StatusOK, LoginResponse{
		Token:            res.Token,
		User:             res.User,
		MFASetupRequired: res.MFASetupRequired,
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

func (h *Handler) setDeviceCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     deviceCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		MaxAge:   int(trustedDeviceTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

type VerifyLoginRequest struct {
	MFAToken       string `json:"mfaToken"`
	Code           string `json:"code"`
	RememberDevice bool   `json:"rememberDevice"`
}

// VerifyLogin handles POST /api/v1/auth/login/2fa — the code step.
func (h *Handler) VerifyLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[VerifyLoginRequest](w, r)
	if !ok {
		return
	}
	token, user, err := h.service.VerifyLogin(r.Context(), req.MFAToken, req.Code)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	if req.RememberDevice {
		if dt, err := h.service.NewTrustedDevice(r.Context(), user.ID, r.UserAgent()); err == nil {
			h.setDeviceCookie(w, dt)
		}
	}
	h.setSessionCookie(w, token)
	response.JSON(w, http.StatusOK, LoginResponse{Token: token, User: user})
}

// TwoFactorStatus handles GET /api/v1/auth/2fa
func (h *Handler) TwoFactorStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	st, err := h.service.TwoFactorStatus(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, st)
}

// BeginSetup handles POST /api/v1/auth/2fa/setup
func (h *Handler) BeginSetup(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	setup, err := h.service.BeginSetup(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.JSON(w, http.StatusOK, setup)
}

type codeRequest struct {
	Code string `json:"code"`
}

// ConfirmSetup handles POST /api/v1/auth/2fa/enable {code}
func (h *Handler) ConfirmSetup(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	req, ok := decodeJSON[codeRequest](w, r)
	if !ok {
		return
	}
	codes, token, err := h.service.ConfirmSetup(r.Context(), id, req.Code)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	// Replace a setup-only session with a full one.
	h.setSessionCookie(w, token)
	w.Header().Set("Cache-Control", "no-store")
	response.JSON(w, http.StatusOK, map[string]any{"backupCodes": codes})
}

// RegenerateBackupCodes handles POST /api/v1/auth/2fa/backup-codes {code}
func (h *Handler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	req, ok := decodeJSON[codeRequest](w, r)
	if !ok {
		return
	}
	codes, err := h.service.RegenerateBackupCodes(r.Context(), id, req.Code)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.JSON(w, http.StatusOK, map[string]any{"backupCodes": codes})
}

// DisableTwoFactor handles POST /api/v1/auth/2fa/disable {code}
func (h *Handler) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	req, ok := decodeJSON[codeRequest](w, r)
	if !ok {
		return
	}
	if err := h.service.DisableTwoFactor(r.Context(), id, req.Code); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ForgetDevices handles POST /api/v1/auth/2fa/forget-devices
func (h *Handler) ForgetDevices(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	if err := h.service.ForgetDevices(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ResetTwoFactor handles POST /api/v1/auth/2fa/reset/{userId} (admins).
func (h *Handler) ResetTwoFactor(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFromContext(r.Context())
	target, err := parseID(chi.URLParam(r, "userId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.service.ResetTwoFactor(r.Context(), id, target); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
