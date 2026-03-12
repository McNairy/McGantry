package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
)

type contextKey string

const claimsKey contextKey = "claims"
const effectiveRoleKey contextKey = "effectiveRole"

// RequireAuth returns middleware that validates the Bearer token in the
// Authorization header. It accepts both JWT tokens and long-lived API keys
// (tokens prefixed with "gantry_"). Claims are stored in the request context.
// The effective role (accounting for group memberships) is also computed and stored.
func RequireAuth(authSvc *auth.Service, database *db.DB) func(http.Handler) http.Handler {
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

			token := parts[1]
			var claims *auth.Claims

			if strings.HasPrefix(token, auth.APIKeyPrefix) {
				// Validate as an API key.
				keyHash := auth.HashAPIKey(token)
				apiKey, err := database.GetAPIKeyByHash(r.Context(), keyHash)
				if err != nil {
					http.Error(w, `{"error":"invalid or revoked api key"}`, http.StatusUnauthorized)
					return
				}
				// Synthesize claims from the stored API key.
				claims = &auth.Claims{
					UserID:   apiKey.UserID,
					Username: "apikey:" + apiKey.Name,
					Role:     apiKey.Role,
				}
			} else {
				// Validate as a JWT.
				var err error
				claims, err = authSvc.ValidateToken(token)
				if err != nil {
					http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
					return
				}
			}

			// Compute effective role from user's direct role + group roles.
			effectiveRole := claims.Role
			if claims.UserID != "" {
				groups, err := database.ListUserGroups(r.Context(), claims.UserID)
				if err == nil && len(groups) > 0 {
					groupRoles := make([]string, len(groups))
					for i, g := range groups {
						groupRoles[i] = g.Role
					}
					effectiveRole = auth.EffectiveRole(claims.Role, groupRoles)
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, effectiveRoleKey, effectiveRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks whether the authenticated user's
// effective role (accounting for group memberships) meets the minimum required
// role according to the hierarchy: admin > platform-engineer > developer > viewer.
func RequireRole(role string) func(http.Handler) http.Handler {
	requiredLevel := auth.RoleLevel(role)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			effective := GetEffectiveRole(r.Context())
			userLevel := auth.RoleLevel(effective)
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

// GetEffectiveRole returns the effective role from the request context.
// Falls back to the claims role if no effective role is set.
func GetEffectiveRole(ctx context.Context) string {
	if role, ok := ctx.Value(effectiveRoleKey).(string); ok && role != "" {
		return role
	}
	if claims := GetClaims(ctx); claims != nil {
		return claims.Role
	}
	return ""
}
