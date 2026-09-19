package bookings

import (
	"context"
	"errors"
	"testing"

	"github.com/odysight/crm/pkg/response"
)

type fakeSiteGuard struct {
	owner map[int64]int64 // siteID -> customerID
	calls int
}

func (f *fakeSiteGuard) EnsureOwnedBy(_ context.Context, siteID, customerID int64) error {
	f.calls++
	owner, ok := f.owner[siteID]
	if !ok {
		return response.NewAPIError(404, "site not found")
	}
	if owner != customerID {
		return response.NewAPIError(422, "site belongs to a different customer")
	}
	return nil
}

func ptr(v int64) *int64 { return &v }

func TestAssertSiteAllowed(t *testing.T) {
	guard := &fakeSiteGuard{owner: map[int64]int64{10: 1, 20: 2}}
	svc := &Service{sites: guard}
	ctx := context.Background()

	t.Run("no site is always allowed", func(t *testing.T) {
		// The one-off flow books with neither a customer nor a site.
		if err := svc.assertSiteAllowed(ctx, nil, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := svc.assertSiteAllowed(ctx, nil, ptr(1)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("site belonging to the customer is allowed", func(t *testing.T) {
		if err := svc.assertSiteAllowed(ctx, ptr(10), ptr(1)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("another customer's site is rejected", func(t *testing.T) {
		err := svc.assertSiteAllowed(ctx, ptr(20), ptr(1))
		if err == nil {
			t.Fatal("expected a cross-customer site to be rejected")
		}
		var apiErr *response.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 422 {
			t.Fatalf("got %v, want a 422 API error", err)
		}
	})

	t.Run("site without a customer is rejected before reaching the guard", func(t *testing.T) {
		before := guard.calls
		err := svc.assertSiteAllowed(ctx, ptr(10), nil)
		if err == nil {
			t.Fatal("expected siteId without customerId to be rejected")
		}
		var apiErr *response.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 422 {
			t.Fatalf("got %v, want a 422 API error", err)
		}
		if guard.calls != before {
			t.Fatal("guard consulted despite there being no customer to check against")
		}
	})

	t.Run("unknown site is rejected", func(t *testing.T) {
		if err := svc.assertSiteAllowed(ctx, ptr(999), ptr(1)); err == nil {
			t.Fatal("expected an unknown site to be rejected")
		}
	})
}
