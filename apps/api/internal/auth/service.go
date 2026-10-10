package auth

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/odysight/crm/pkg/response"
)

const tokenTTLDefault = 8 * time.Hour

// Lockout policy: after MaxFailed attempts for an email, block for Window.
const (
	maxFailedLogins = 5
	lockoutWindow   = 15 * time.Minute
)

type Service struct {
	repo      *Repository
	jwtSecret []byte
	tokenTTL  time.Duration
	twoFA     twoFactorKeys

	mu           sync.Mutex
	failedLogins map[string]int
	lockouts     map[string]time.Time
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return NewServiceWithTTL(repo, jwtSecret, tokenTTLDefault)
}

func NewServiceWithTTL(repo *Repository, jwtSecret string, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = tokenTTLDefault
	}
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  ttl,
		twoFA:     newTwoFactorKeys([]byte(jwtSecret)),
		// fresh on startup; lockouts are an extra layer over the IP rate limiter
		failedLogins: make(map[string]int),
		lockouts:     make(map[string]time.Time),
	}
}

func (s *Service) isLocked(email string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if until, ok := s.lockouts[email]; ok {
		if time.Now().Before(until) {
			return true
		}
		delete(s.lockouts, email)
		delete(s.failedLogins, email)
	}
	return false
}

func (s *Service) recordFailure(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failedLogins[email]++
	if s.failedLogins[email] >= maxFailedLogins {
		s.lockouts[email] = time.Now().Add(lockoutWindow)
		delete(s.failedLogins, email)
	}
}

func (s *Service) clearFailures(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failedLogins, email)
	delete(s.lockouts, email)
}

// Login checks the password. deviceToken is the "remember this device"
// cookie, which lets a trusted device skip the 2FA code.
func (s *Service) Login(ctx context.Context, req LoginRequest, deviceToken string) (LoginResult, error) {
	if err := req.Validate(); err != nil {
		return LoginResult{}, err
	}

	email := req.Email
	if s.isLocked(email) {
		return LoginResult{}, response.NewAPIError(429, "too many failed login attempts; try again later")
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			s.recordFailure(email)
			return LoginResult{}, response.NewAPIError(401, "invalid email or password")
		}
		return LoginResult{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.recordFailure(email)
		return LoginResult{}, response.NewAPIError(401, "invalid email or password")
	}

	s.clearFailures(email)
	return s.afterPassword(ctx, user, deviceToken)
}

func (s *Service) Me(ctx context.Context, id int64) (UserDTO, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return UserDTO{}, response.NewAPIError(401, "invalid token subject")
	}
	return toDTO(u), nil
}

func (s *Service) ChangePassword(ctx context.Context, id int64, oldPass, newPass string) error {
	if len(newPass) < 8 {
		return response.NewAPIError(400, "new password must be at least 8 characters")
	}
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return response.NewAPIError(401, "invalid token subject")
	}
	full, err := s.repo.GetByEmail(ctx, u.Email)
	if err != nil {
		return response.NewAPIError(401, "invalid token subject")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(full.PasswordHash), []byte(oldPass)); err != nil {
		return response.NewAPIError(400, "current password is incorrect")
	}
	hash, err := HashPassword(newPass)
	if err != nil {
		return response.NewAPIError(500, "failed to hash password")
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

// signToken issues a session. setupOnly marks an office user who still has
// to enrol in 2FA: such a session only reaches the /auth endpoints.
func (s *Service) signToken(u User, setupOnly bool) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  strconv.FormatInt(u.ID, 10),
		"role": string(u.Role),
		"iss":  "odysight-crm",
		"aud":  "odysight-web",
		"iat":  now.Unix(),
		"exp":  now.Add(s.tokenTTL).Unix(),
	}
	if setupOnly {
		claims[claimMFASetup] = true
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", response.NewAPIError(500, "failed to issue token")
	}
	return signed, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
