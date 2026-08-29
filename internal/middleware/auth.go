package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nikhil-O1O5/url-shortener/internal/service"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func RequireAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := extractClaims(r, authService)
			if err != nil || claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, _ := extractClaims(r, authService)
			if claims != nil {
				ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) *string {
	val, ok := ctx.Value(UserIDKey).(string)
	if !ok || val == "" {
		return nil
	}
	return &val
}

func extractClaims(r *http.Request, authService *service.AuthService) (*service.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, nil
	}

	return authService.ValidateToken(parts[1])
}
