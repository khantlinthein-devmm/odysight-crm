package attendance

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Location is a phone's reported position; Accuracy is the GPS error radius
// in metres (0 when unknown).
type Location struct {
	Lat      float64
	Lng      float64
	Accuracy float64
}

// SiteCoord is a job site with known coordinates.
type SiteCoord struct {
	ID   int64
	Name string
	Lat  float64
	Lng  float64
}

// stamp is what a check-in/out records about where it happened.
type stamp struct {
	loc      *Location
	siteID   *int64
	distance *int
}

func (s stamp) lat() any {
	if s.loc == nil {
		return nil
	}
	return s.loc.Lat
}

func (s stamp) lng() any {
	if s.loc == nil {
		return nil
	}
	return s.loc.Lng
}

// distanceMeters is the great-circle (haversine) distance between two points.
func distanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(a)))
}

// nearestSite returns the closest site to loc and its distance.
func nearestSite(loc Location, sites []SiteCoord) (SiteCoord, float64) {
	best, bestD := SiteCoord{}, math.Inf(1)
	for _, s := range sites {
		if d := distanceMeters(loc.Lat, loc.Lng, s.Lat, s.Lng); d < bestD {
			best, bestD = s, d
		}
	}
	return best, bestD
}

// allowedRadius widens the configured radius by the phone's reported GPS
// error (capped at 100 m) so indoor fixes near the site are not rejected.
func allowedRadius(radius int, accuracy float64) float64 {
	return float64(radius) + math.Min(math.Max(accuracy, 0), 100)
}

func formatDistance(m float64) string {
	if m >= 1000 {
		return fmt.Sprintf("%.1f km", m/1000)
	}
	return fmt.Sprintf("%.0f m", m)
}

// SitesForCleanerOn lists the coordinates of sites where the cleaner has an
// active job on the local calendar day of date.
func (r *Repository) SitesForCleanerOn(ctx context.Context, cleanerID int64, date time.Time) ([]SiteCoord, error) {
	y, m, d := date.Local().Date()
	from := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT s.id, s.name, s.latitude, s.longitude
		   FROM bookings b
		   JOIN sites s ON s.id = b.site_id
		  WHERE s.latitude IS NOT NULL AND s.longitude IS NOT NULL
		    AND b.status IN ('pending', 'confirmed', 'in_progress', 'completed')
		    AND b.scheduled_for >= $2 AND b.scheduled_for < $3
		    AND (EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = b.id AND bc.cleaner_id = $1)
		         OR b.assigned_cleaner = (SELECT trim(first_name || ' ' || last_name) FROM cleaners WHERE id = $1))`,
		cleanerID, from, from.AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("sites for cleaner %d: %w", cleanerID, err)
	}
	defer rows.Close()
	var out []SiteCoord
	for rows.Next() {
		var s SiteCoord
		if err := rows.Scan(&s.ID, &s.Name, &s.Lat, &s.Lng); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
