package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/pkg/errors"
)

var ErrUnauthorized = errors.New("unauthorized")

type Introspector interface {
	Introspect(ctx context.Context, token string) (*accessor.Claims, error)
}

func NewIntrospectMiddleware(introspector Introspector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				http.Error(resp, "authorization header invalid", http.StatusUnauthorized)
				return
			}

			claims, err := introspector.Introspect(req.Context(), token)
			if err != nil {
				err = errors.Wrap(err, "introspection failure")
				slog.Error("introspection failure", "error", err.Error())

				if errors.Is(err, ErrUnauthorized) {
					http.Error(resp, err.Error(), http.StatusUnauthorized)
					return
				}
				http.Error(resp, err.Error(), http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(req.Context(), accessor.UserClaims, claims)
			next.ServeHTTP(resp, req.WithContext(ctx))
		})
	}
}
