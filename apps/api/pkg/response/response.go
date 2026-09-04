package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("encode response", "error", err)
		}
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorBody{Error: message})
}

// HandleError maps domain errors to HTTP responses.
// Unknown errors are logged and returned as 500 without internals.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		Error(w, apiErr.Status, apiErr.Message)
		return
	}
	slog.ErrorContext(r.Context(), "internal error", "path", r.URL.Path, "error", err)
	Error(w, http.StatusInternalServerError, "internal server error")
}
