package bookings

import (
	"net/http/httptest"
	"testing"

	"github.com/odysight/crm/internal/auth"
)

func TestDtoForHidesPriceFromCleaners(t *testing.T) {
	price := 1850.0
	b := Booking{ID: 1, Price: &price}

	for _, tc := range []struct {
		role      auth.Role
		wantPrice bool
	}{
		{auth.RoleCleaner, false},
		{auth.RoleDispatch, true},
		{auth.RoleAdmin, true},
	} {
		r := httptest.NewRequest("GET", "/bookings/1", nil)
		r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{UserID: 9, Role: tc.role}))
		got := dtoFor(r, b)
		if (got.Price != nil) != tc.wantPrice {
			t.Errorf("%s: price visible = %v, want %v", tc.role, got.Price != nil, tc.wantPrice)
		}
	}
}
