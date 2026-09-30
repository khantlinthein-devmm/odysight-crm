package attendance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysight/crm/internal/auth"
)

// checkRequest runs a check-in through checkScope + allowSelf as user 7.
func checkRequest(t *testing.T, h *Handler, role auth.Role, body CheckActionRequest) (int, int64) {
	t.Helper()
	var self int64
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		self, _ = selfCleaner(r)
		if allowSelf(w, r, body) {
			w.WriteHeader(http.StatusOK)
		}
	})
	req := httptest.NewRequest(http.MethodPost, "/check-in", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{UserID: 7, Role: role}))
	rec := httptest.NewRecorder()
	h.checkScope(next).ServeHTTP(rec, req)
	return rec.Code, self
}

func stubHandler(cleanerID int64, err error) *Handler {
	return &Handler{cleanerForUser: func(context.Context, int64) (int64, error) { return cleanerID, err }}
}

func TestAdminChecksInAnyone(t *testing.T) {
	for _, body := range []CheckActionRequest{
		{PersonType: PersonCleaner, PersonID: 99},
		{PersonType: PersonStaff, PersonID: 55},
	} {
		if code, _ := checkRequest(t, stubHandler(0, ErrNoCleanerProfile), auth.RoleAdmin, body); code != http.StatusOK {
			t.Fatalf("admin for %+v: got %d, want 200", body, code)
		}
	}
}

func TestOfficeStaffOnlyThemselves(t *testing.T) {
	// The bug report: a signed-in user could check in someone else.
	for _, role := range []auth.Role{auth.RoleDispatch, auth.RoleManager, auth.RoleAccountant} {
		h := stubHandler(0, ErrNoCleanerProfile)
		if code, _ := checkRequest(t, h, role, CheckActionRequest{PersonType: PersonStaff, PersonID: 7}); code != http.StatusOK {
			t.Fatalf("%s checking themselves in: got %d, want 200", role, code)
		}
		if code, _ := checkRequest(t, h, role, CheckActionRequest{PersonType: PersonStaff, PersonID: 8}); code != http.StatusForbidden {
			t.Fatalf("%s checking in another staff member: got %d, want 403", role, code)
		}
		if code, _ := checkRequest(t, h, role, CheckActionRequest{PersonType: PersonCleaner, PersonID: 99}); code != http.StatusForbidden {
			t.Fatalf("%s checking in a cleaner: got %d, want 403", role, code)
		}
	}
}

func TestCleanerOnlyOwnProfile(t *testing.T) {
	h := stubHandler(42, nil)
	code, self := checkRequest(t, h, auth.RoleCleaner, CheckActionRequest{PersonType: PersonCleaner, PersonID: 42})
	if code != http.StatusOK || self != 42 {
		t.Fatalf("cleaner on own profile: code=%d self=%d", code, self)
	}
	if code, _ := checkRequest(t, h, auth.RoleCleaner, CheckActionRequest{PersonType: PersonCleaner, PersonID: 43}); code != http.StatusForbidden {
		t.Fatalf("cleaner checking in another cleaner: got %d, want 403", code)
	}
	if code, _ := checkRequest(t, h, auth.RoleCleaner, CheckActionRequest{PersonType: PersonStaff, PersonID: 7}); code != http.StatusForbidden {
		t.Fatalf("cleaner checking in as staff: got %d, want 403", code)
	}
}

func TestCleanerWithoutProfileIsRefused(t *testing.T) {
	if code, _ := checkRequest(t, stubHandler(0, ErrNoCleanerProfile), auth.RoleCleaner,
		CheckActionRequest{PersonType: PersonCleaner, PersonID: 1}); code != http.StatusForbidden {
		t.Fatalf("unlinked cleaner: got %d, want 403", code)
	}
}

func TestListScopePinsCleanerToSelf(t *testing.T) {
	var self int64
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		self, _ = selfCleaner(r)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{UserID: 7, Role: auth.RoleCleaner}))
	stubHandler(42, nil).scope(auth.PermAttendanceRead)(next).ServeHTTP(httptest.NewRecorder(), req)
	if !reached || self != 42 {
		t.Fatalf("cleaner list: reached=%v self=%d", reached, self)
	}
}
