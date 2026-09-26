package complaints

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/odysight/crm/internal/bookings"
	"github.com/odysight/crm/pkg/response"
)

// BookingCreator books the re-clean visit (bookings.Service in production),
// reusing all booking validation, crew-conflict checks and notifications.
type BookingCreator interface {
	Create(ctx context.Context, req bookings.CreateBookingRequest) (bookings.Booking, error)
}

type Service struct {
	repo     *Repository
	bookings BookingCreator
	now      func() time.Time
}

func NewService(repo *Repository, creator BookingCreator) *Service {
	return &Service{repo: repo, bookings: creator, now: time.Now}
}

func mapErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "complaint not found")
	}
	return err
}

func (s *Service) List(ctx context.Context, f Filters) ([]Complaint, int, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id int64) (Complaint, error) {
	c, err := s.repo.Get(ctx, id)
	return c, mapErr(err)
}

type CreateRequest struct {
	CustomerID  *int64 `json:"customerId"`
	SiteID      *int64 `json:"siteId"`
	BookingID   *int64 `json:"bookingId"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Channel     string `json:"channel"`
	Description string `json:"description"`
}

func (s *Service) Create(ctx context.Context, req CreateRequest, userID int64) (Complaint, error) {
	req.Category = strings.TrimSpace(req.Category)
	req.Severity = strings.TrimSpace(req.Severity)
	req.Channel = strings.TrimSpace(req.Channel)
	req.Description = strings.TrimSpace(req.Description)
	if !categories[req.Category] {
		return Complaint{}, response.NewAPIError(400, "category must be one of quality, missed_area, late, no_show, damage, staff_behaviour, other")
	}
	if _, ok := slaHours[req.Severity]; !ok {
		return Complaint{}, response.NewAPIError(400, "severity must be low, medium or high")
	}
	if req.Channel == "" {
		req.Channel = "phone"
	}
	if !channels[req.Channel] {
		return Complaint{}, response.NewAPIError(400, "channel must be phone, line, email, portal or in_person")
	}
	if req.Description == "" {
		return Complaint{}, response.NewAPIError(400, "description is required")
	}

	c := Complaint{Category: req.Category, Severity: req.Severity, Channel: req.Channel,
		Description: req.Description, SiteID: req.SiteID, BookingID: req.BookingID}
	if req.BookingID != nil {
		custID, siteID, err := s.repo.BookingOwner(ctx, *req.BookingID)
		if errors.Is(err, ErrNotFound) {
			return Complaint{}, response.NewAPIError(400, "booking not found")
		}
		if err != nil {
			return Complaint{}, err
		}
		if custID == nil {
			return Complaint{}, response.NewAPIError(400, "that booking has no customer record; pick the customer instead")
		}
		if req.CustomerID != nil && *req.CustomerID != *custID {
			return Complaint{}, response.NewAPIError(400, "booking belongs to a different customer")
		}
		req.CustomerID = custID
		if c.SiteID == nil {
			c.SiteID = siteID
		}
	}
	if req.CustomerID == nil || *req.CustomerID < 1 {
		return Complaint{}, response.NewAPIError(400, "customerId or bookingId is required")
	}
	c.CustomerID = *req.CustomerID
	if c.SiteID != nil {
		owner, err := s.repo.SiteCustomer(ctx, *c.SiteID)
		if errors.Is(err, ErrNotFound) {
			return Complaint{}, response.NewAPIError(400, "site not found")
		}
		if err != nil {
			return Complaint{}, err
		}
		if owner != c.CustomerID {
			return Complaint{}, response.NewAPIError(400, "site belongs to a different customer")
		}
	}
	c.DueAt = DueAt(c.Severity, s.now())
	created, err := s.repo.Create(ctx, c, userID)
	if err != nil && strings.Contains(err.Error(), "complaints_customer_id_fkey") {
		return Complaint{}, response.NewAPIError(400, "customer not found")
	}
	return created, err
}

type UpdateRequest struct {
	Status      *string `json:"status"`
	Severity    *string `json:"severity"`
	Category    *string `json:"category"`
	Description *string `json:"description"`
	Resolution  *string `json:"resolution"`
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest, userID int64) (Complaint, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Complaint{}, mapErr(err)
	}
	var p Patch
	if req.Status != nil {
		st := Status(strings.TrimSpace(*req.Status))
		if !st.Valid() {
			return Complaint{}, response.NewAPIError(400, "status must be open, in_progress, resolved or closed")
		}
		p.Status = &st
	}
	if req.Severity != nil {
		sev := strings.TrimSpace(*req.Severity)
		if _, ok := slaHours[sev]; !ok {
			return Complaint{}, response.NewAPIError(400, "severity must be low, medium or high")
		}
		p.Severity = &sev
		// A changed severity moves the deadline, measured from when it was raised.
		due := DueAt(sev, current.CreatedAt)
		p.DueAt = &due
	}
	if req.Category != nil {
		cat := strings.TrimSpace(*req.Category)
		if !categories[cat] {
			return Complaint{}, response.NewAPIError(400, "invalid category")
		}
		p.Category = &cat
	}
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		if d == "" {
			return Complaint{}, response.NewAPIError(400, "description cannot be empty")
		}
		p.Description = &d
	}
	if req.Resolution != nil {
		r := strings.TrimSpace(*req.Resolution)
		p.Resolution = &r
	}
	resolution := current.Resolution
	if p.Resolution != nil {
		resolution = *p.Resolution
	}
	if p.Status != nil && p.Status.Done() && resolution == "" {
		return Complaint{}, response.NewAPIError(400, "write what was done (resolution) before resolving")
	}
	updated, err := s.repo.Update(ctx, id, p, userID)
	return updated, mapErr(err)
}

type RecleanRequest struct {
	ScheduledFor    string  `json:"scheduledFor"`
	DurationMinutes int     `json:"durationMinutes"`
	CleanerIDs      []int64 `json:"cleanerIds"`
	// SameCrew re-sends the original job's crew when CleanerIDs is empty.
	SameCrew bool `json:"sameCrew"`
}

// Reclean books a free follow-up visit for the complaint and links it.
func (s *Service) Reclean(ctx context.Context, id int64, req RecleanRequest) (Complaint, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Complaint{}, mapErr(err)
	}
	if c.Status.Done() {
		return Complaint{}, response.NewAPIError(409, "complaint is already resolved")
	}
	if c.RecleanBookingID != nil {
		return Complaint{}, response.NewAPIError(409, "a re-clean is already booked: "+c.RecleanNumber)
	}
	var src SourceBooking
	if c.BookingID != nil {
		src, err = s.repo.SourceBooking(ctx, *c.BookingID)
	} else {
		src, err = s.repo.CustomerBasics(ctx, c.CustomerID)
		src.ServiceType, src.DurationMinutes = "reclean", 120
	}
	if err != nil {
		return Complaint{}, mapErr(err)
	}
	crew := req.CleanerIDs
	if len(crew) == 0 && req.SameCrew {
		crew = src.CleanerIDs
	}
	duration := src.DurationMinutes
	if req.DurationMinutes > 0 {
		duration = req.DurationMinutes
	}
	customerID := c.CustomerID
	b, err := s.bookings.Create(ctx, bookings.CreateBookingRequest{
		CustomerName:    src.CustomerName,
		CustomerEmail:   src.CustomerEmail,
		CustomerID:      &customerID,
		SiteID:          c.SiteID,
		ServiceType:     src.ServiceType,
		ScheduledFor:    req.ScheduledFor,
		DurationMinutes: duration,
		Address:         src.Address,
		Area:            src.Area,
		Status:          "confirmed",
		Notes:           fmt.Sprintf("Free re-clean for complaint %s: %s", c.Number, truncate(c.Description, 200)),
		CleanerIDs:      crew,
	})
	if err != nil {
		return Complaint{}, err
	}
	return s.repo.LinkReclean(ctx, id, b.ID)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
