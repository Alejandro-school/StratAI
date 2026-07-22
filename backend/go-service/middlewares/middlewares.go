package middlewares

import (
	"net/http"
	"os"
	"strings"
)

// getAllowedOrigins returns the list of allowed CORS origins from env or defaults.
func getAllowedOrigins() []string {
	raw := os.Getenv("ALLOWED_ORIGINS")
	if raw != "" {
		origins := strings.Split(raw, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		return origins
	}
	return []string{
		"http://localhost:3000",
		"http://localhost:8000",
		"http://127.0.0.1:3000",
	}
}

// isOriginAllowed checks if the request origin is in the allowed list.
func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

// WithCors applies CORS headers using a whitelist approach.
func WithCors(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithSecurityHeaders adds OWASP-recommended security headers.
func WithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		next.ServeHTTP(w, r)
	})
}

// WithInternalOnly restricts access to localhost (inter-service communication only).
func WithInternalOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		// Extract IP from host:port
		if idx := strings.LastIndex(remoteAddr, ":"); idx >= 0 {
			remoteAddr = remoteAddr[:idx]
		}
		// Strip brackets from IPv6
		remoteAddr = strings.Trim(remoteAddr, "[]")

		if remoteAddr != "127.0.0.1" && remoteAddr != "::1" && remoteAddr != "localhost" {
			http.Error(w, "Forbidden: internal endpoint only", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
