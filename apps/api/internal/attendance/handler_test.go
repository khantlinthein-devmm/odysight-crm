package attendance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysight/crm/internal/auth"
)

func scopedRequest(t *testing.T, h *Handler, p auth.Permission, role auth.Role, body CheckActionRequest) (int, bool, int64) {
	t.Helper()
	var reached bool
	var self int64
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		self, _ = selfCleaner(r)
		if allowSelf(w, r, body) {
			w.WriteHeader(http.StatusOK)
		}
	})
	req := httptest.NewRequest(http.MethodPost, "/check-in", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{UserID: 7, Role: role}))
	rec := httptest.NewRecorder()
	h.scope(p)(next).ServeHTTP(rec, req)
	return rec.Code, reached, self
}

func stubHandler(cleanerID int64, err error) *Handler {
	return &Handler{cleanerForUser: func(context.Context, int64) (int64, error) { return cleanerID, err }}
}

func TestScopeStaffWithPermissionActsOnAnyone(t *testing.T) {
	code, reached, self := scopedRequest(t, stubHandler(0, ErrNoCleanerProfile), auth.PermAttendanceManage, auth.RoleDispatch,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 99})
	if code != http.StatusOK || !reached || self != 0 {
		t.Fatalf("dispatch should act on any cleaner: code=%d reached=%v self=%d", code, reached, self)
	}
}

func TestScopeCleanerChecksInThemselves(t *testing.T) {
	code, _, self := scopedRequest(t, stubHandler(42, nil), auth.PermAttendanceManage, auth.RoleCleaner,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 42})
	if code != http.StatusOK || self != 42 {
		t.Fatalf("cleaner should check in themselves: code=%d self=%d", code, self)
	}
}

func TestScopeCleanerCannotCheckInSomeoneElse(t *testing.T) {
	code, _, _ := scopedRequest(t, stubHandler(42, nil), auth.PermAttendanceManage, auth.RoleCleaner,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 43})
	if code != http.StatusForbidden {
		t.Fatalf("cleaner checking in another cleaner: got %d, want 403", code)
	}
	code, _, _ = scopedRequest(t, stubHandler(42, nil), auth.PermAttendanceManage, auth.RoleCleaner,
		CheckActionRequest{PersonType: PersonStaff, PersonID: 42})
	if code != http.StatusForbidden {
		t.Fatalf("cleaner checking in staff: got %d, want 403", code)
	}
}

func TestScopeCleanerWithoutProfileIsRefused(t *testing.T) {
	code, reached, _ := scopedRequest(t, stubHandler(0, ErrNoCleanerProfile), auth.PermAttendanceManage, auth.RoleCleaner,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 1})
	if code != http.StatusForbidden || reached {
		t.Fatalf("unlinked cleaner: code=%d reached=%v, want 403 and blocked", code, reached)
	}
}

func TestScopeOtherRolesWithoutPermissionAreRefused(t *testing.T) {
	code, reached, _ := scopedRequest(t, stubHandler(42, nil), auth.PermAttendanceManage, auth.RoleAccountant,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 42})
	if code != http.StatusForbidden || reached {
		t.Fatalf("accountant: code=%d reached=%v, want 403", code, reached)
	}
}
