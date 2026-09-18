package portal

import (
	"context"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/odysight/crm/internal/feedback"
	"github.com/odysight/crm/pkg/response"
)

const tokenTTLDefault = 8 * time.Hour

type Service struct {
	repo      *Repository
	feedback  *feedback.Service
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