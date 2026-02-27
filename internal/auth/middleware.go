package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const (
	claimsKey   contextKey = "auth_claims"
	subjectKey  contextKey = "auth_subject"
)

// ClaimsFromContext extracts JWT claims from request context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

// SubjectFromContext extracts the subject ID string from request context.
func SubjectFromContext(ctx context.Context) string {
	sub, _ := ctx.Value(subjectKey).(string)
	return sub
}

// AuthenticatePlayer returns middleware that validates player JWT tokens.
func AuthenticatePlayer(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return authenticateRealm(jwtMgr, RealmPlayer)
}

// AuthenticateAdmin returns middleware that validates admin JWT tokens.
func AuthenticateAdmin(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return authenticateRealm(jwtMgr, RealmAdmin)
}

// AuthenticateAffiliate returns middleware that validates affiliate JWT tokens
// and checks for suspension.
func AuthenticateAffiliate(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := extractAndValidate(r, jwtMgr, RealmAffiliate)
			if err != nil {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			// Affiliate-specific: check suspension
			if claims.Status == "suspended" {
				http.Error(w, `{"code":"FORBIDDEN","message":"affiliate account suspended"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, subjectKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks the admin role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"no auth context"}`, http.StatusUnauthorized)
				return
			}
			if !roleSet[claims.Role] {
				http.Error(w, `{"code":"FORBIDDEN","message":"insufficient role"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authenticateRealm(jwtMgr *JWTManager, realm Realm) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := extractAndValidate(r, jwtMgr, realm)
			if err != nil {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, subjectKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractAndValidate(r *http.Request, jwtMgr *JWTManager, realm Realm) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fmt.Errorf("invalid Authorization format")
	}

	return jwtMgr.ValidateTokenForRealm(parts[1], realm)
}
