package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

type LoginResponse struct {
	Token string  `json:"token,omitempty"`
	User  UserDTO `json:"user"`
	// MFARequired: send the app code with MFAToken to /auth/login/2fa.
	MFARequired bool   `json:"mfaRequired,omitempty"`
	MFAToken    string `json:"mfaToken,omitempty"`
	// MFASetupRequired: the session only works for 2FA setup until done.
	MFASetupRequired bool `json:"mfaSetupRequired,omitempty"`
}

func Routes(h *Handler, az *Authorizer, loginLimit func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	if loginLimit == nil {
		loginLimit = func(next http.Handler) http.Handler { return next }
	}
	r.With(loginLimit).Post("/login", h.Login)
	r.With(loginLimit).Post("/login/2fa", h.VerifyLogin)
	if az != nil {
		r.Group(func(r chi.Router) {
			r.Use(az.Authenticate)
			r.Get("/me", h.Me)
			r.Post("/logout", h.Logout)
			r.With(loginLimit).Post("/change-password", h.ChangePassword)
			r.Get("/2fa", h.TwoFactorStatus)
			r.Post("/2fa/setup", h.BeginSetup)
			r.With(loginLimit).Post("/2fa/enable", h.ConfirmSetup)
			r.With(loginLimit).Post("/2fa/backup-codes", h.RegenerateBackupCodes)
			r.With(loginLimit).Post("/2fa/disable", h.DisableTwoFactor)
			r.Post("/2fa/forget-devices", h.ForgetDevices)
			r.With(az.Require(PermUsersManage)).Post("/2fa/reset/{userId}", h.ResetTwoFactor)
		})
	}
	return r
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
