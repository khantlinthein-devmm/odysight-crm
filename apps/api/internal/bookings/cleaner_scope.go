package bookings

import (
	"context"
	"errors"
	"strings"

	"github.com/odysight/crm/pkg/response"
)

// CLEANER logins hold bookings.read/update so the mobile app works, but they
// must only see and touch their own jobs (plus the unassigned pool they can
// accept from). These helpers narrow those requests; staff are unaffected.

// cleanerFor resolves the caller's cleaner profile. ok is false when the
// login has no linked profile.
func (s *Service) cleanerFor(ctx context.Context, userID int64) (CleanerIdentity, bool, error) {
	c, err := s.repo.FindCleanerForUser(ctx, userID)
	if errors.Is(err, ErrUnknownCleaner) {
		return CleanerIdentity{}, false, nil
	}
	if err != nil {
		return CleanerIdentity{}, false, err
	}
	return c, true, nil
}

// assignedTo reports whether c is on b's crew.
func assignedTo(b Booking, c CleanerIdentity) bool {
	for _, x := range b.Cleaners {
		if x.ID == c.ID {
			return true
		}
	}
	return len(b.Cleaners) == 0 && c.Name != "" && strings.EqualFold(strings.TrimSpace(b.AssignedCleaner), c.Name)
}

// inPool reports whether b is an unassigned pending job any cleaner may view
// and accept.
func inPool(b Booking) bool {
	return b.Status == StatusPending && strings.TrimSpace(b.AssignedCleaner) == "" && len(b.Cleaners) == 0
}

// checkCleanerUpdate allows a cleaner to move their own job to in_progress or
// completed and to add notes; every other field stays staff-only.
func checkCleanerUpdate(b Booking, c CleanerIdentity, req UpdateBookingRequest) error {
	if !assignedTo(b, c) {
		return response.NewAPIError(403, "you can only update jobs assigned to you")
	}
	if req.CustomerName != nil || req.CustomerEmail != nil || req.CustomerID != nil ||
		req.SiteID != nil || req.ContractID != nil || req.ClearSiteID != nil || req.ClearContractID != nil ||
		req.ServiceType != nil || req.ScheduledFor != nil || req.DurationMinutes != nil ||
		req.Address != nil || req.Area != nil || req.AssignedCleaner != nil ||
		req.IsRecurring != nil || req.Recurrence != nil || req.CleanerIDs != nil {
		return response.NewAPIError(403, "cleaners can only change a job's status or notes")
	}
	if req.Status != nil {
		switch Status(strings.TrimSpace(*req.Status)) {
		case StatusInProgress, StatusCompleted:
		default:
			return response.NewAPIError(403, "cleaners can only mark a job in progress or completed")
		}
	}
	return nil
}
