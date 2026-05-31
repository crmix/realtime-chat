package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
)

type ctxKey int

const userCtxKey ctxKey = iota

// Auth returns middleware that requires a valid bearer token. The resolved
// user ID is attached to the request context for downstream handlers.
func Auth(parser interface {
	ParseToken(string) (uuid.UUID, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error())
				return
			}
			id, err := parser.ParseToken(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error())
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user ID from a request context populated
// by Auth middleware. Returns uuid.Nil when not present.
func UserID(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(userCtxKey).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

// TokenFromRequest exposes bearer-or-query token extraction so non-HTTP
// transports (WebSocket upgrade) can reuse the same convention.
func TokenFromRequest(r *http.Request) string {
	if t := bearerToken(r); t != "" {
		return t
	}
	return r.URL.Query().Get("token")
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
