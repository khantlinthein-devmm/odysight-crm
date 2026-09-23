package portal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/odysight/crm/internal/feedback"
	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/pkg/response"
)

const tokenTTLDefault = 8 * time.Hour

type Service struct {
	repo      *Repository
	feedback  *feedback.Service
	notifier  *notifications.Service
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(repo *Repository, feedbackSvc *feedback.Service, jwtSecret string) *Service {
	return NewServiceWithTTL(repo, feedbackSvc, jwtSecret, tokenTTLDefault)
}

func NewServiceWithTTL(repo *Repository, feedbackSvc *feedback.Service, jwtSecret string, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = tokenTTLDefault
	}
	return &Service{repo: repo, feedback: feedbackSvc, jwtSecret: []byte(jwtSecret), tokenTTL: ttl}
}

// WithNotifier attaches the office notification service. Kept as a setter so
// existing constructors and tests keep compiling; portal self-bookings notify
// the office only when a notifier is attached.
func (s *Service) WithNotifier(n *notifications.Service) *Service {
	s.notifier = n
	return s
}

// LoginRequest is the payload for the portal login endpoint.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Token is the returned portal session.
type Token struct {
	Token    string   `json:"token"`
	Customer Customer `json:"customer"`
}

func (r LoginRequest) valid() bool {
	return r.Email != "" && r.Password != ""
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (string, Customer, error) {
	if !req.valid() {
		return "", Customer{}, response.NewAPIError(400, "email and password are required")
	}
	c, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return "", Customer{}, response.NewAPIError(401, "invalid email or password")
	}
	if !c.PortalEnabled || c.Status == "blocked" {
		return "", Customer{}, response.NewAPIError(403, "portal access is not enabled for this account")
	}
	if c.PasswordHash == nil || *c.PasswordHash == "" {
		return "", Customer{}, response.NewAPIError(403, "no portal password set; contact the office")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*c.PasswordHash), []byte(req.Password)); err != nil {
		return "", Customer{}, response.NewAPIError(401, "invalid email or password")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": strconv.FormatInt(c.ID, 10),
		"iss": "odysight-crm",
		"aud": portalAudience,
		"iat": now.Unix(),
		"exp": now.Add(s.tokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", Customer{}, response.NewAPIError(500, "failed to issue token")
	}
	return signed, Customer{
		ID:      c.ID,
		Name:    c.Name,
		Email:   c.Email,
		Phone:   c.Phone,
		Address: c.Address,
		Area:    c.Area,
	}, nil
}

// requireAccess re-checks that a customer may still use the portal, so
// disabling portal access or blocking the account takes effect immediately
// instead of lingering until the JWT expires.
func (s *Service) requireAccess(ctx context.Context, customerID int64) error {
	enabled, blocked, err := s.repo.AccessState(ctx, customerID)
	if err != nil {
		return response.NewAPIError(401, "invalid token subject")
	}
	if !enabled || blocked {
		return response.NewAPIError(403, "portal access is not enabled for this account")
	}
	return nil
}

// Me returns the portal customer's profile.
func (s *Service) Me(ctx context.Context, customerID int64) (Customer, error) {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return Customer{}, err
	}
	c, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return Customer{}, response.NewAPIError(401, "invalid token subject")
	}
	return c, nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, customerID int64, current, newPass string) error {
	if len(newPass) < 8 {
		return response.NewAPIError(400, "new password must be at least 8 characters")
	}
	if err := s.requireAccess(ctx, customerID); err != nil {
		return err
	}
	hash, err := s.repo.PasswordHashByID(ctx, customerID)
	if err != nil {
		return response.NewAPIError(401, "invalid token subject")
	}
	if hash == "" {
		return response.NewAPIError(403, "no portal password set; contact the office")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)); err != nil {
		return response.NewAPIError(400, "current password is incorrect")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return response.NewAPIError(500, "failed to hash password")
	}
	if err := s.repo.SetPassword(ctx, customerID, string(newHash)); err != nil {
		return err
	}
	return nil
}

// Bookings lists the customer's own bookings.
func (s *Service) Bookings(ctx context.Context, customerID int64) ([]Booking, error) {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return nil, err
	}
	return s.repo.Bookings(ctx, customerID)
}

// Sites lists the customer's own active service locations.
func (s *Service) Sites(ctx context.Context, customerID int64) ([]Site, error) {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListSites(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Site{}
	}
	return items, nil
}

// Services returns the bookable service catalog for the portal.
func (s *Service) Services(ctx context.Context, customerID int64) ([]ServiceItem, error) {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return nil, err
	}
	items, err := s.repo.ActiveServices(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ServiceItem{}
	}
	return items, nil
}

// CreateBooking creates a pending booking for the logged-in customer.
// Past dates and unknown service types are rejected; site ownership is
// enforced in the repository. No contract or cleaner is ever assigned here.
func (s *Service) CreateBooking(ctx context.Context, customerID int64, req CreateBookingRequest) (Booking, error) {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return Booking{}, err
	}
	req.ServiceType = strings.TrimSpace(req.ServiceType)
	req.Address = strings.TrimSpace(req.Address)
	req.Notes = strings.TrimSpace(req.Notes)
	req.ScheduledFor = strings.TrimSpace(req.ScheduledFor)
	if req.ServiceType == "" {
		return Booking{}, response.NewAPIError(400, "serviceType is required")
	}
	if req.ScheduledFor == "" {
		return Booking{}, response.NewAPIError(400, "scheduledFor is required")
	}
	scheduledFor, err := time.Parse(time.RFC3339, req.ScheduledFor)
	if err != nil {
		return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
	}
	if scheduledFor.Before(time.Now().Add(-5 * time.Minute)) {
		return Booking{}, response.NewAPIError(400, "scheduledFor must be in the future")
	}
	duration := req.DurationMinutes
	if duration == 0 {
		duration = 120
	}
	if duration < 15 || duration > 24*60 {
		return Booking{}, response.NewAPIError(400, "durationMinutes must be 15..1440")
	}
	b, err := s.repo.CreatePortalBooking(ctx, customerID, req, scheduledFor, duration)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "site does not belong") || strings.Contains(msg, "site not found") {
			return Booking{}, response.NewAPIError(422, msg)
		}
		return Booking{}, err
	}
	s.notifyOfficeNewBooking(b)
	return b, nil
}

// notifyOfficeNewBooking tells the office about a portal self-booking in the
// background, so slow delivery never blocks the customer response. Delivery
// (or the skip reason) is always written to the notification log, which the
// office can watch in the notification center.
func (s *Service) notifyOfficeNewBooking(b Booking) {
	if s.notifier == nil {
		return
	}
	go func() {
		ctx := context.Background()
		subject := fmt.Sprintf("New portal booking %s — %s", b.BookingNumber, b.ServiceType)
		body := `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
			`<h2 style="margin-bottom:4px;">New customer portal booking</h2>` +
			`<p style="color:#64748b;margin-top:0;">` + b.BookingNumber + ` · pending confirmation</p>` +
			`<table style="width:100%;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;">` +
			`<tr><td style="padding:8px 12px;">Service</td><td style="padding:8px 12px;font-weight:600;">` + b.ServiceType + `</td></tr>` +
			`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Scheduled</td><td style="padding:8px 12px;">` + b.ScheduledFor.Format("Mon Jan 2 at 15:04") + `</td></tr>` +
			`<tr><td style="padding:8px 12px;">Address</td><td style="padding:8px 12px;">` + b.Address + `</td></tr>` +
			`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Notes</td><td style="padding:8px 12px;">` + b.Notes + `</td></tr>` +
			`</table>` +
			`<p style="color:#94a3b8;font-size:12px;margin-top:24px;">Confirm and assign a cleaner from Bookings → Dispatch.</p>` +
			`</div>`
		s.notifier.NotifyOffice(ctx, notifications.EventBookingCreated, subject, body)
	}()
}

// CancelBooking cancels the customer's own pending/confirmed booking.
func (s *Service) CancelBooking(ctx context.Context, customerID, bookingID int64) error {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return err
	}
	if err := s.repo.CancelPortalBooking(ctx, customerID, bookingID); err != nil {
		if err == ErrNotFound {
			return response.NewAPIError(404, "booking not found or cannot be cancelled")
		}
		return err
	}
	return nil
}

// SubmitFeedback records portal feedback for one of the customer's own
// completed bookings. Returns the feedback id and booking number as a slug.
func (s *Service) SubmitFeedback(ctx context.Context, customerID int64, bookingID int64, rating int, comment string) error {
	if err := s.requireAccess(ctx, customerID); err != nil {
		return err
	}
	status, err := s.repo.EnsureBookingOwned(ctx, customerID, bookingID)
	if err != nil {
		return response.NewAPIError(404, "booking not found")
	}
	if status != "completed" {
		return response.NewAPIError(400, "feedback can only be submitted for completed bookings")
	}
	req := feedback.CreateFeedbackRequest{
		BookingID: bookingID,
		Rating:    rating,
		Comment:   comment,
	}
	cid := customerID
	_, err = s.feedback.Create(ctx, req, &cid)
	if err != nil {
		return err
	}
	return nil
}