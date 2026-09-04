package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/odysight/crm/pkg/response"
)

const tokenTTL = 8 * time.Hour

type Service struct {
	repo      *Repository
	jwtSecret []byte
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: []byte(jwtSecret)}
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (string, UserDTO, error) {
	if err := req.Validate(); err != nil {
		return "", UserDTO{}, err
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return "", UserDTO{}, response.NewAPIError(401, "invalid email or password")
		}
		return "", UserDTO{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", UserDTO{}, response.NewAPIError(401, "invalid email or password")
	}

	token, err := s.signToken(user)
	if err != nil {
		return "", UserDTO{}, err
	}
	return token, toDTO(user), nil
}

func (s *Service) signToken(u User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  u.ID,
		"role": string(u.Role),
		"exp":  time.Now().Add(tokenTTL).Unix(),
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
