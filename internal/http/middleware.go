package http

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds security-related headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Swagger UI needs scripts, styles, and images to render
		if strings.HasPrefix(r.URL.Path, "/swagger/") {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
		} else {
			w.Header().Set("Content-Security-Policy", "default-src 'none'")
		}

		next.ServeHTTP(w, r)
	})
}
