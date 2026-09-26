package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
	// cleanerForUser resolves the cleaner profile linked to a login. It is a
	// field so tests can stub it without a database.
	cleanerForUser func(ctx context.Context, userID int64) (int64, error)
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, cleanerForUser: service.CleanerIDForUser}
}

type selfKey struct{}

// scope admits callers holding p unchanged. A CLEANER without p is admitted
// in self-service mode, pinned to their own cleaner profile; anyone else is
// refused.
func (h *Handler) scope(p auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := auth.IdentityFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if auth.HasPermission(id.Role, p) {
				next.ServeHTTP(w, r)
				return
			}
			if id.Role != auth.RoleCleaner {
				response.Error(w, http.StatusForbidden, "permission denied: "+string(p))
				return
			}
			cleanerID, err := h.cleanerForUser(r.Context(), id.UserID)
			if errors.Is(err, ErrNoCleanerProfile) {
				response.Error(w, http.StatusForbidden, "your login is not linked to a cleaner profile; ask an admin")
				return
			}
			if err != nil {
				response.HandleError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), selfKey{}, cleanerID)))
		})
	}
}

// selfCleaner returns the caller's own cleaner id when the request runs in
// self-service mode.
func selfCleaner(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(selfKey{}).(int64)
	return id, ok
}

// List handles GET /api/v1/attendance?type=&person=&from=&to=&limit=&offset=.
// type is cleaner or staff (omitted means both) and person is that workforce's
// id. from/to are work dates (YYYY-MM-DD); pagination.Parse only accepts
// RFC3339 timestamps, so they are read off the query string here instead.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 50, nil)
	from, ok := parseDateParam(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseDateParam(w, r, "to")
	if !ok {
		return
	}
	personType, ok := parsePersonType(w, r)
	if !ok {
		return
	}
	filters := Filters{
		PersonType: personType,
		PersonID:   parsePersonID(r),
		From:       from,
		To:         to,
	}
	if self, ok := selfCleaner(r); ok {
		filters.PersonType = PersonCleaner
		filters.PersonID = self
	}
	items, total, err := h.service.List(r.Context(), filters, params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]RecordDTO, 0, len(items))
	for _, rec := range items {
		dtos = append(dtos, toDTO(rec))
	}
	response.JSON(w, http.StatusOK, pagination.Page[RecordDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// CheckIn handles POST /api/v1/attendance/check-in.
func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CheckActionRequest](w, r)
	if !ok {
		return
	}
	if !allowSelf(w, r, req) {
		return
	}
	rec, err := h.service.CheckIn(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(rec))
}

// CheckOut handles POST /api/v1/attendance/check-out.
func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CheckActionRequest](w, r)
	if !ok {
		return
	}
	if !allowSelf(w, r, req) {
		return
	}
	rec, err := h.service.CheckOut(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(rec))
}

// allowSelf rejects a self-service request that targets anyone other than
// the caller's own cleaner profile.
func allowSelf(w http.ResponseWriter, r *http.Request, req CheckActionRequest) bool {
	self, ok := selfCleaner(r)
	if !ok {
		return true
	}
	pt := req.PersonType
	if pt == "" {
		pt = PersonCleaner
	}
	if pt != PersonCleaner || req.PersonID != self {
		response.Error(w, http.StatusForbidden, "cleaners can only record their own attendance")
		return false
	}
	return true
}

// parsePersonType reads ?type=cleaner|staff. An empty value means both.
func parsePersonType(w http.ResponseWriter, r *http.Request) (PersonType, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("type"))
	if raw == "" {
		return "", true
	}
	pt := PersonType(raw)
	if !pt.Valid() {
		response.Error(w, http.StatusBadRequest, "type must be cleaner or staff")
		return "", false
	}
	return pt, true
}

// parsePersonID reads ?person=, falling back to the legacy ?cleaner= filter.
func parsePersonID(r *http.Request) int64 {
	raw := strings.TrimSpace(r.URL.Query().Get("person"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("cleaner"))
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0
	}
	return id
}

func parseDateParam(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return "", true
	}
	if _, err := time.Parse(dateFormat, raw); err != nil {
		response.Error(w, http.StatusBadRequest, key+" must be a YYYY-MM-DD date")
		return "", false
	}
	return raw, true
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
