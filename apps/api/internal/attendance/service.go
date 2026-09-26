package attendance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// dateFormat is the YYYY-MM-DD layout used for attendance work dates.
const dateFormat = "2006-01-02"

type Service struct {
	repo *Repository
	// radius returns the check-in geofence in metres (0 = off).
	radius func(ctx context.Context) int
	now    func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo, radius: func(context.Context) int { return 0 }, now: time.Now}
}

// WithGeofence sets where the check-in radius comes from (Settings →
// Booking defaults).
func (s *Service) WithGeofence(radius func(ctx context.Context) int) *Service {
	s.radius = radius
	return s
}

// CleanerIDForUser returns the cleaner profile linked to a login.
func (s *Service) CleanerIDForUser(ctx context.Context, userID int64) (int64, error) {
	return s.repo.CleanerIDForUser(ctx, userID)
}

func (s *Service) List(ctx context.Context, f Filters, params pagination.Params) ([]Record, int, error) {
	items, total, err := s.repo.List(ctx, f, params)
	if err != nil {
		return nil, 0, mapRepoError(err)
	}
	return items, total, nil
}

// CheckIn stamps the person's arrival on the current work date. The date is
// taken from the server clock, so the API container's TZ decides the rollover
// hour (docker-compose sets Asia/Bangkok).
// A cleaner checking themselves in (selfService) must be near one of the
// day's job sites when geofencing is on and those sites have coordinates.
func (s *Service) CheckIn(ctx context.Context, req CheckActionRequest, selfService bool) (Record, error) {
	if err := req.Validate(); err != nil {
		return Record{}, err
	}
	at := stamp{loc: req.Location()}
	if req.PersonType == PersonCleaner {
		if err := s.locate(ctx, req.PersonID, &at, selfService); err != nil {
			return Record{}, err
		}
	}
	rec, err := s.repo.CheckIn(ctx, req.Person(), today(), at)
	if err != nil {
		return Record{}, mapRepoError(err)
	}
	return rec, nil
}

// CheckOut stamps the person's departure on the current work date.
func (s *Service) CheckOut(ctx context.Context, req CheckActionRequest) (Record, error) {
	if err := req.Validate(); err != nil {
		return Record{}, err
	}
	rec, err := s.repo.CheckOut(ctx, req.Person(), today(), stamp{loc: req.Location()})
	if err != nil {
		return Record{}, mapRepoError(err)
	}
	return rec, nil
}

func today() string { return time.Now().Format(dateFormat) }

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, ErrAlreadyCheckedIn):
		return response.NewAPIError(409, "already checked in for today")
	case errors.Is(err, ErrAlreadyCheckedOut):
		return response.NewAPIError(409, "already checked out for today")
	}
	return dberror.Map(err, ErrNotFound, "attendance record not found")
}

// locate fills in the nearest job site and distance for a cleaner check-in,
// and (for self-service) enforces the geofence radius.
func (s *Service) locate(ctx context.Context, cleanerID int64, at *stamp, enforce bool) error {
	radius := s.radius(ctx)
	if at.loc == nil && (!enforce || radius <= 0) {
		return nil
	}
	sites, err := s.repo.SitesForCleanerOn(ctx, cleanerID, s.now())
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		// No job site with coordinates today: nothing to check against.
		return nil
	}
	if at.loc == nil {
		return response.NewAPIError(422, "location is required to check in; allow location access on your phone")
	}
	site, d := nearestSite(*at.loc, sites)
	meters := int(d + 0.5)
	at.siteID, at.distance = &site.ID, &meters
	if enforce && radius > 0 && d > allowedRadius(radius, at.loc.Accuracy) {
		return response.NewAPIError(403, fmt.Sprintf(
			"you are %s from %s; check in within %d m of the site", formatDistance(d), site.Name, radius))
	}
	return nil
}
