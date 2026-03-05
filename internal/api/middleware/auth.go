package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gantrydev/gantry/internal/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

// roleHierarchy defines the authorization level for each role.
// Higher values imply greater privilege.
var roleHierarchy = map[string]int{
	"viewer":            1,
	"developer":         2,
	"platform-engineer": 3,
	"admin":             4,
}

// RequireAuth returns middleware that validates the Bearer token in the
// Authorization header and stores the decoded claims in the request context.
func RequireAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks whether the authenticated user's
// role meets the minimum required role according to the hierarchy:
// admin > platform-engineer > developer > viewer.
func RequireRole(role string) func(http.Handler) http.Handler {
	requiredLevel := roleHierarchy[role]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			userLevel := roleHierarchy[claims.Role]
			if userLevel < requiredLevel {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims extracts the auth.Claims from the request context.
// It returns nil if no claims are present.
func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}
