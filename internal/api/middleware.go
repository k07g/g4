package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/k07g/g4/internal/auth"
)

type contextKey string

const (
	contextKeyAccessToken contextKey = "accessToken"
	contextKeyCognitoSub  contextKey = "cognitoSub"
	contextKeyEmail       contextKey = "email"
)

// AuthMiddleware validates the bearer access token on protected routes by
// resolving it against Cognito, and attaches the caller's identity to the
// request context.
func AuthMiddleware(provider auth.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			identity, err := provider.GetUser(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired access token")
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyAccessToken, token)
			ctx = context.WithValue(ctx, contextKeyCognitoSub, identity.Sub)
			ctx = context.WithValue(ctx, contextKeyEmail, identity.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
