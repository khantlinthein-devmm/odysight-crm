package bookings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/internal/sites"
	"github.com/odysight/crm/pkg/response"
)

// This guards the exact regression a live check caught while building the
// site picker: BookingForm.vue submits the full form — including siteId — on
// every save, edit included. The PATCH handler decodes UpdateBookingRequest
// with DisallowUnknownFields, so if that struct doesn't carry a SiteID field,
// every booking edit breaks the moment the frontend starts sending it, not
// just ones that touch the site.
func TestUpdateBookingRequestAcceptsSiteID(t *testing.T) {
	body := []byte(`{"siteId": 5}`)
	var req UpdateBookingRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		t.Fatalf("UpdateBookingRequest rejected a siteId field: %v", err)
	}
	if req.SiteID == nil || *req.SiteID != 5 {
		t.Fatalf("siteId did not decode: got %+v", req)
	}
}

func seedCustomerForBooking(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status)
		 VALUES ($1, '', '', '0800000000', 'Test address', 'other', 'Test area', 'active')
		 RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, id)
	})
	return id
}

func TestDBUpdateBookingSiteOwnership(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	siteSvc := sites.NewService(sites.NewRepository(pool))
	svc := NewService(repo, (*notifications.Service)(nil), siteSvc)
	ctx := context.Background()

	stamp := time.Now().Format("150405.000000")
	customerA := seedCustomerForBooking(t, pool, "Update Owner A "+stamp)
	customerB := seedCustomerForBooking(t, pool, "Update Owner B "+stamp)

	siteA, err := siteSvc.Create(ctx, sites.CreateSiteRequest{CustomerID: customerA, Name: "A site"})
	if err != nil {
		t.Fatalf("create site for A: %v", err)
	}

	b, err := repo.Create(ctx, Booking{
		CustomerName:    "Update Test",
		CustomerID:      &customerB,
		ServiceType:     "office",
		ScheduledFor:    time.Now().Add(48 * time.Hour),
		DurationMinutes: 120,
		Status:          StatusPending,
	})
	if err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM bookings WHERE id = $1`, b.ID)
	})

	// Booking belongs to customer B; siteA belongs to customer A.
	_, err = svc.Update(ctx, b.ID, UpdateBookingRequest{SiteID: &siteA.ID})
	if err == nil {
		t.Fatal("expected a cross-customer site update to be rejected")
	}
	var apiErr *response.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 422 {
		t.Fatalf("got %v, want a 422 API error", err)
	}

	// Moving the booking to customer A in the same request makes siteA valid.
	updated, err := svc.Update(ctx, b.ID, UpdateBookingRequest{CustomerID: &customerA, SiteID: &siteA.ID})
	if err != nil {
		t.Fatalf("update with matching customer+site: %v", err)
	}
	if updated.SiteID == nil || *updated.SiteID != siteA.ID {
		t.Fatalf("siteId not persisted: got %+v", updated.SiteID)
	}
	if updated.CustomerID == nil || *updated.CustomerID != customerA {
		t.Fatalf("customerId not persisted: got %+v", updated.CustomerID)
	}
}
