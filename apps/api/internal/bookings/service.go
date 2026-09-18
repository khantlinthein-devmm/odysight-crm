package bookings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo     *Repository
	notifier *notifications.Service
}

func NewService(repo *Repository, notifier *notifications.Service) *Service {
	return &Service{repo: repo, notifier: notifier}
}

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Booking, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Booking, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	return b, nil
}

func (s *Service) Create(ctx context.Context, req CreateBookingRequest) (Booking, error) {
	if err := req.Validate(); err != nil {
		return Booking{}, err
	}

	scheduledFor, err := time.Parse(time.RFC3339, req.ScheduledFor)
	if err != nil {
		return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
	}

	endTime := scheduledFor.Add(time.Duration(req.DurationMinutes) * time.Minute)
	cleaners, err := s.resolveCleaners(ctx, validateCleanerIDs(req.CleanerIDs))
	if err != nil {
		return Booking{}, err
	}
	if err := s.assertNoConflicts(ctx, cleanerIDs(cleaners), scheduledFor, endTime, 0); err != nil {
		return Booking{}, err
	}

	b := Booking{
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerID:      req.CustomerID,
		ServiceType:     ServiceType(req.ServiceType),
		ScheduledFor:    scheduledFor,
		DurationMinutes: req.DurationMinutes,
		Address:         req.Address,
		Area:            req.Area,
		AssignedCleaner: req.AssignedCleaner,
		Status:          Status(req.Status),
		Notes:           req.Notes,
		IsRecurring:     req.IsRecurring,
		Recurrence:      req.Recurrence,
		Cleaners:        cleaners,
	}
	if req.IsRecurring {
		seriesID := newSeriesID()
		b.SeriesID = &seriesID
	}

	created, err := s.repo.Create(ctx, b)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	s.notifyCreated(created)
	return created, nil
}

// notifyCreated sends booking-confirmation notifications to the customer on
// email, SMS and WhatsApp best-effort and in the background, so slow delivery
// never blocks the API response. Unconfigured channels are logged as skipped.
func (s *Service) notifyCreated(b Booking) {
	if s.notifier == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if strings.TrimSpace(b.CustomerEmail) != "" {
			s.notifier.EmitEmail(ctx, notifications.EventBookingCreated, b.CustomerEmail,
				"Your cleaning appointment is confirmed",
				bookingConfirmationHTML(b))
		}
		phone := ""
		if b.CustomerID != nil {
			phone, _ = s.repo.CustomerPhone(ctx, *b.CustomerID)
		}
		phone = strings.TrimSpace(phone)
		if phone == "" {
			return
		}
		text := bookingConfirmationText(b)
		s.notifier.EmitSMS(ctx, notifications.ChannelSMS, notifications.EventBookingCreated, phone, text)
		s.notifier.EmitSMS(ctx, notifications.ChannelWhatsApp, notifications.EventBookingCreated, phone, text)
	}()
}

func bookingConfirmationText(b Booking) string {
	return fmt.Sprintf("Hi %s, your booking %s is confirmed for %s (%s). Service: %s. Thank you for choosing Smile Clean!",
		b.CustomerName, b.BookingNumber, b.ScheduledFor.Format("Mon Jan 2 at 15:04"), b.Status, b.ServiceType)
}

func bookingConfirmationHTML(b Booking) string {
	return `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
		`<h2 style="margin-bottom:4px;">Your booking is confirmed</h2>` +
		`<p style="color:#64748b;margin-top:0;">` + b.BookingNumber + `</p>` +
		`<table style="width:100%;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;">` +
		`<tr><td style="padding:8px 12px;">Customer</td><td style="padding:8px 12px;font-weight:600;">` + htmlEscape(b.CustomerName) + `</td></tr>` +
		`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Service</td><td style="padding:8px 12px;font-weight:600;">` + htmlEscape(string(b.ServiceType)) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;">Scheduled</td><td style="padding:8px 12px;">` + b.ScheduledFor.Format("Mon Jan 2 at 15:04") + `</td></tr>` +
		`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Address</td><td style="padding:8px 12px;">` + htmlEscape(b.Address) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;">Status</td><td style="padding:8px 12px;">` + htmlEscape(string(b.Status)) + `</td></tr>` +
		`</table>` +
		`<p style="color:#94a3b8;font-size:12px;margin-top:24px;">Thank you for choosing Smile Clean!</p>` +
		`</div>`
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(s)
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateBookingRequest) (Booking, error) {
	if err := req.Validate(); err != nil {
		return Booking{}, err
	}
	if req.IsEmpty() {
		return Booking{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.CustomerName != nil {
		v := strings.TrimSpace(*req.CustomerName)
		patch.CustomerName = &v
	}
	if req.CustomerEmail != nil {
		v := strings.ToLower(strings.TrimSpace(*req.CustomerEmail))
		patch.CustomerEmail = &v
	}
	if req.CustomerID != nil {
		patch.CustomerID = req.CustomerID
	}
	if req.ServiceType != nil {
		st := ServiceType(*req.ServiceType)
		patch.ServiceType = &st
	}
	if req.ScheduledFor != nil {
		t, err := time.Parse(time.RFC3339, *req.ScheduledFor)
		if err != nil {
			return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
		}
		patch.ScheduledFor = &t
	}
	if req.DurationMinutes != nil {
		patch.DurationMinutes = req.DurationMinutes
	}
	if req.Address != nil {
		v := strings.TrimSpace(*req.Address)
		patch.Address = &v
	}
	if req.Area != nil {
		v := strings.TrimSpace(*req.Area)
		patch.Area = &v
	}
	if req.AssignedCleaner != nil {
		v := strings.TrimSpace(*req.AssignedCleaner)
		patch.AssignedCleaner = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}
	if req.IsRecurring != nil {
		recurring := *req.IsRecurring
		patch.IsRecurring = &recurring
	}
	if req.Recurrence != nil {
		v := strings.TrimSpace(*req.Recurrence)
		patch.Recurrence = &v
	}

	if req.CleanerIDs != nil {
		ids := validateCleanerIDs(req.CleanerIDs)
		cleaners, err := s.resolveCleaners(ctx, ids)
		if err != nil {
			return Booking{}, err
		}
		patch.Cleaners = cleaners
	}

	// Resolve the resulting schedule + cleaners and surface conflicts BEFORE
	// persisting anything, so a rejected change leaves no partial write behind.
	changedSchedule := req.ScheduledFor != nil || req.DurationMinutes != nil
	if req.CleanerIDs != nil || changedSchedule {
		current, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return Booking{}, mapRepoError(err)
		}
		start := current.ScheduledFor
		duration := current.DurationMinutes
		if req.ScheduledFor != nil {
			t, err := time.Parse(time.RFC3339, *req.ScheduledFor)
			if err != nil {
				return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
			}
			start = t
		}
		if req.DurationMinutes != nil {
			duration = *req.DurationMinutes
		}
		checking := cleanerIDs(current.Cleaners)
		if req.CleanerIDs != nil {
			checking = cleanerIDs(patch.Cleaners)
		}
		if err := s.assertNoConflicts(ctx, checking, start, start.Add(time.Duration(duration)*time.Minute), id); err != nil {
			return Booking{}, err
		}
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	return updated, nil
}

// validateCleanerIDs dedupes and orders a raw ID slice, dropping zero values.
func validateCleanerIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// resolveCleaners validates that every cleaner ID exists and wraps them with
// "primary"/"crew" roles.
func (s *Service) resolveCleaners(ctx context.Context, ids []int64) ([]CleanerBrief, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	names, err := s.repo.CleanerNames(ctx, ids)
	if err != nil {
		if err == ErrUnknownCleaner {
			return nil, response.NewAPIError(422, "one or more assigned cleaners do not exist")
		}
		return nil, err
	}
	byID := map[int64]string{}
	for _, n := range names {
		byID[n.ID] = n.Name
	}
	out := make([]CleanerBrief, 0, len(ids))
	for i, id := range ids {
		role := "crew"
		if i == 0 {
			role = "primary"
		}
		out = append(out, CleanerBrief{ID: id, Name: byID[id], Role: role})
	}
	return out, nil
}

func cleanerIDs(cleaners []CleanerBrief) []int64 {
	ids := make([]int64, 0, len(cleaners))
	for _, c := range cleaners {
		ids = append(ids, c.ID)
	}
	return ids
}

// assertNoConflicts returns a structured 409 listing each overlapping booking.
func (s *Service) assertNoConflicts(ctx context.Context, ids []int64, start, end time.Time, excludeID int64) error {
	if len(ids) == 0 {
		return nil
	}
	conflicts, err := s.repo.FindConflicts(ctx, ids, start, end, excludeID)
	if err != nil {
		return err
	}
	if len(conflicts) == 0 {
		return nil
	}
	parts := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		parts = append(parts, fmt.Sprintf("%s (%s) %s %s", c.CleanerName, c.BookingNumber, c.CustomerName, c.ScheduledFor.Format("Jan 2 15:04")))
	}
	return response.NewAPIError(409, "Cleaner is already booked: "+strings.Join(parts, "; "))
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

// Available lists the unassigned pool for the logged-in cleaner, scoped to
// the cleaner's own area unless an explicit area filter is given. A cleaner
// with no area set (or an explicit ?area=) sees across areas.
func (s *Service) Available(ctx context.Context, userID int64, params pagination.Params) ([]Booking, int, string, error) {
	cleaner, err := s.repo.FindCleanerForUser(ctx, userID)
	if err != nil {
		if err == ErrUnknownCleaner {
			return nil, 0, "", response.NewAPIError(404, "no cleaner profile linked to this login; ask the office to link your account")
		}
		return nil, 0, "", err
	}
	area := strings.TrimSpace(params.Area)
	if area == "" {
		area = strings.TrimSpace(cleaner.Area)
	}
	params.Area = area
	params.Available = true
	params.Status = ""
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, "", err
	}
	return items, total, area, nil
}

// Accept lets the logged-in cleaner take a pending booking. First tap wins.
func (s *Service) Accept(ctx context.Context, userID, bookingID int64) (Booking, error) {
	cleaner, err := s.repo.FindCleanerForUser(ctx, userID)
	if err != nil {
		if err == ErrUnknownCleaner {
			return Booking{}, response.NewAPIError(404, "no cleaner profile linked to this login; ask the office to link your account")
		}
		return Booking{}, err
	}
	existing, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	if existing.Area != "" && cleaner.Area != "" &&
		!strings.EqualFold(strings.TrimSpace(existing.Area), strings.TrimSpace(cleaner.Area)) {
		return Booking{}, response.NewAPIError(403, "booking is outside your area")
	}
	end := existing.ScheduledFor.Add(time.Duration(existing.DurationMinutes) * time.Minute)
	if err := s.assertNoConflicts(ctx, []int64{cleaner.ID}, existing.ScheduledFor, end, bookingID); err != nil {
		return Booking{}, err
	}
	b, err := s.repo.Accept(ctx, bookingID, cleaner)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	return b, nil
}

func mapRepoError(err error) error {
	if err == ErrAlreadyAssigned {
		return response.NewAPIError(409, "booking is no longer available")
	}
	return dberror.Map(err, ErrNotFound, "booking not found")
}
