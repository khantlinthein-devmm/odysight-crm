package portal

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
)

// sessionCookieName is the HttpOnly cookie holding a customer-portal session.
const sessionCookieName = "odysight_portal_session"

// Authorizer authenticates customer-portal tokens (audience odysight-portal).
type Authorizer struct {
	jwtSecret []byte
}

func NewAuthorizer(jwtSecret string) *Authorizer {
	return &Authorizer{jwtSecret: []byte(jwtSecret)}
}

// Authenticate validates the portal JWT and stores the customer id in context.
func (a *Authorizer) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := portalTokenFromRequest(r)
		if token == "" {
			response.Error(w, http.StatusUnauthorized, "missing authentication")
			return
		}
		parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return a.jwtSecret, nil
		}, jwt.WithIssuer("odysight-crm"), jwt.WithAudience(portalAudience), jwt.WithExpirationRequired())
		if err != nil || !parsed.Valid {
			response.Error(w, http.StatusUnauthorized, "invalid or expired portal session")
			return
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "invalid token claims")
			return
		}
		sub, err := claims.GetSubject()
		if err != nil || sub == "" {
			response.Error(w, http.StatusUnauthorized, "invalid token subject")
			return
		}
		var customerID int64
		for i := 0; i < len(sub); i++ {
			c := sub[i]
			if c < '0' || c > '9' {
				response.Error(w, http.StatusUnauthorized, "invalid token subject")
				return
			}
			customerID = customerID*10 + int64(c-'0')
			if customerID > 1<<62 {
				response.Error(w, http.StatusUnauthorized, "invalid token subject")
				return
			}
		}
		if customerID <= 0 || len(sub) == 0 {
			response.Error(w, http.StatusUnauthorized, "invalid token subject")
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPortalCustomerID(r.Context(), customerID)))
	})
}

func portalTokenFromRequest(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		if len(header) > 7 && header[:7] == "Bearer " {
			return header[7:]
		}
	}
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

// CustomerID extracts the portal customer id previously stored by
// Authenticate.
func CustomerID(r *http.Request) int64 {
	id, _ := auth.PortalCustomerIDFromContext(r.Context())
	return id
}