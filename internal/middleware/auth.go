package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
)

func InternalSecret(secret string) func(http.Handler) http.Handler {
	expected := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(r.Header.Get("X-Internal-Secret"))
			if len(got) == 0 || subtle.ConstantTimeCompare(got, expected) != 1 {
				slog.Warn("unauthorized request",
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
