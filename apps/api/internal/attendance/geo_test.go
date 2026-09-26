package attendance

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/response"
)

func TestDistanceMeters(t *testing.T) {
	// Two points in Siam about 500 m apart.
	d := distanceMeters(13.7462, 100.5347, 13.7466, 100.5393)
	if d < 450 || d > 550 {
		t.Fatalf("distance = %.0f m, want ~500", d)
	}
	if distanceMeters(13.7, 100.5, 13.7, 100.5) != 0 {
		t.Fatal("same point should be 0")
	}
}

func TestAllowedRadiusCapsAccuracy(t *testing.T) {
	if got := allowedRadius(200, 30); got != 230 {
		t.Errorf("allowedRadius(200,30) = %v", got)
	}
	if got := allowedRadius(200, 5000); got != 300 {
		t.Errorf("accuracy should cap at 100 m, got %v", got)
	}
}

func TestNearestSite(t *testing.T) {
	sites := []SiteCoord{{ID: 1, Lat: 13.80, Lng: 100.55}, {ID: 2, Lat: 13.7463, Lng: 100.5348}}
	s, d := nearestSite(Location{Lat: 13.7462, Lng: 100.5347}, sites)
	if s.ID != 2 || d > 50 {
		t.Fatalf("nearest = %d at %.0f m", s.ID, d)
	}
}

func f(v float64) *float64 { return &v }

// Cleaners checking themselves in must be near today's site; office staff
// checking a cleaner in are not geofenced.
func TestDBGeofencedCheckIn(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var customerID, siteID, cleanerID, bookingID int64
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(pool.QueryRow(ctx, `INSERT INTO customers (first_name, phone, address, property_type, area, status)
		VALUES ('Geo Test', '0800000001', 'x', 'office', 'Siam', 'active') RETURNING id`).Scan(&customerID))
	must(pool.QueryRow(ctx, `INSERT INTO sites (customer_id, name, address, status, latitude, longitude)
		VALUES ($1, 'Paragon', 'Siam', 'active', 13.7462, 100.5347) RETURNING id`, customerID).Scan(&siteID))
	must(pool.QueryRow(ctx, `INSERT INTO cleaners (first_name, last_name, phone, status, area)
		VALUES ('Geo', 'Cleaner', '0800000002', 'available', 'Siam') RETURNING id`).Scan(&cleanerID))
	must(pool.QueryRow(ctx, `INSERT INTO bookings (booking_number, customer_name, customer_id, site_id, service_type,
		scheduled_for, duration_minutes, address, status, assigned_cleaner)
		VALUES ('BK-GEO-' || floor(random() * 1e9)::text, 'Geo Test', $1, $2, 'deep', $3, 60, 'Siam', 'confirmed', '')
		RETURNING id`,
		customerID, siteID, time.Now()).Scan(&bookingID))
	_, err = pool.Exec(ctx, `INSERT INTO booking_cleaners (booking_id, cleaner_id, role) VALUES ($1, $2, 'primary')`, bookingID, cleanerID)
	must(err)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM attendance WHERE cleaner_id = $1`, cleanerID)
		_, _ = pool.Exec(c, `DELETE FROM bookings WHERE id = $1`, bookingID)
		_, _ = pool.Exec(c, `DELETE FROM cleaners WHERE id = $1`, cleanerID)
		_, _ = pool.Exec(c, `DELETE FROM customers WHERE id = $1`, customerID)
	})

	svc := NewService(NewRepository(pool)).WithGeofence(func(context.Context) int { return 200 })
	req := func(lat, lng float64) CheckActionRequest {
		return CheckActionRequest{PersonType: PersonCleaner, PersonID: cleanerID, Latitude: f(lat), Longitude: f(lng), Accuracy: f(10)}
	}

	if _, err := svc.CheckIn(ctx, CheckActionRequest{PersonType: PersonCleaner, PersonID: cleanerID}, true); !isStatus(err, 422) {
		t.Fatalf("no location: err = %v, want 422", err)
	}
	if _, err := svc.CheckIn(ctx, req(13.80, 100.60), true); !isStatus(err, 403) {
		t.Fatalf("far away: err = %v, want 403", err)
	}
	rec, err := svc.CheckIn(ctx, req(13.7463, 100.5349), true)
	if err != nil {
		t.Fatalf("near site: %v", err)
	}
	if rec.CheckInDistanceM == nil || *rec.CheckInDistanceM > 50 || rec.CheckInSiteName != "Paragon" || rec.CheckInLat == nil {
		t.Fatalf("recorded geo = %+v", rec)
	}
	if math.Abs(*rec.CheckInLat-13.7463) > 1e-9 {
		t.Fatalf("lat = %v", *rec.CheckInLat)
	}
}

func isStatus(err error, code int) bool {
	apiErr, ok := err.(*response.APIError)
	return ok && apiErr.Status == code
}
