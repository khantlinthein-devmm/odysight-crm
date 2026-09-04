package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/odysight/crm/pkg/response"
)

type ctxKey int

const identityKey ctxKey = iota

// Permission is a "resource.action" string, e.g. "leads.create".
type Permission string

// Identity holds the authenticated principal extracted from a JWT.
type Identity struct {
	UserID int64
	Role   Role
}

// Authorizer provides JWT authentication and RBAC authorization middleware.
type Authorizer struct {
	jwtSecret []byte
}

func NewAuthorizer(jwtSecret string) *Authorizer {
	return &Authorizer{jwtSecret: []byte(jwtSecret)}
}

// Authenticate validates the Bearer token and stores the Identity in context.
func (a *Authorizer) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			response.Error(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return a.jwtSecret, nil
		})
		if err != nil || !parsed.Valid {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		id, err := claims.GetSubject()
		if err != nil || id == "" {
			response.Error(w, http.StatusUnauthorized, "invalid token subject")
			return
		}
		userID, err := parseID(id)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid token subject")
			return
		}

		rawRole, _ := claims["role"].(string)
		role := Role(rawRole)
		if !role.Valid() {
			response.Error(w, http.StatusForbidden, "unknown role")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey, Identity{
			UserID: userID,
			Role:   role,
		})))
	})
}

// Require returns middleware enforcing the given permission for the current role.
// Must run after Authenticate.
func (a *Authorizer) Require(p Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if !roleHasPermission(identity.Role, p) {
				response.Error(w, http.StatusForbidden, "permission denied: "+string(p))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IdentityFromContext extracts the authenticated Identity, if present.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
}

func parseID(s string) (int64, error) {
	var id int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, jwt.ErrTokenInvalidId
		}
		id = id*10 + int64(c-'0')
		if id > 1<<62 {
			return 0, jwt.ErrTokenInvalidId
		}
	}
	if len(s) == 0 {
		return 0, jwt.ErrTokenInvalidId
	}
	return id, nil
}
