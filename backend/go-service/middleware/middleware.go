package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cs2-demo-service/db"
)

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

const (
	signatureVersion = "v1"
	maxSignatureAge  = 30 * time.Second
	maxInternalBody  = 1024 * 1024
)

func isLocalRequest(r *http.Request) bool {
	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx >= 0 {
		remoteAddr = remoteAddr[:idx]
	}
	remoteAddr = strings.Trim(remoteAddr, "[]")
	return remoteAddr == "127.0.0.1" || remoteAddr == "::1" || remoteAddr == "localhost"
}

func canonicalServiceRequest(r *http.Request, body []byte) string {
	bodyHash := sha256.Sum256(body)
	return strings.Join([]string{
		r.Header.Get("X-Service-Version"),
		strings.ToUpper(r.Method),
		r.URL.RequestURI(),
		r.Header.Get("X-Service-Timestamp"),
		r.Header.Get("X-Service-Nonce"),
		hex.EncodeToString(bodyHash[:]),
	}, "\n")
}

func internalServiceSecret() string {
	if secret := os.Getenv("INTERNAL_SERVICE_SECRET"); secret != "" {
		return secret
	}
	if os.Getenv("APP_ENV") == "production" {
		return ""
	}
	sessionSecret := os.Getenv("SESSION_SECRET_KEY")
	if sessionSecret == "" {
		return ""
	}
	derived := sha256.Sum256([]byte(sessionSecret + ":internal-service"))
	return hex.EncodeToString(derived[:])
}

// WithInternalOnly authenticates service requests with a body-bound HMAC.
func WithInternalOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("APP_ENV") != "production" &&
			os.Getenv("ALLOW_UNSIGNED_LOCAL_INTERNAL") == "true" &&
			isLocalRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		version := r.Header.Get("X-Service-Version")
		timestampValue := r.Header.Get("X-Service-Timestamp")
		nonce := r.Header.Get("X-Service-Nonce")
		signature := r.Header.Get("X-Service-Signature")
		if version != signatureVersion || len(nonce) != 32 || len(signature) != 64 {
			http.Error(w, "invalid service signature", http.StatusForbidden)
			return
		}

		timestamp, err := strconv.ParseInt(timestampValue, 10, 64)
		if err != nil || time.Since(time.Unix(timestamp, 0)).Abs() > maxSignatureAge {
			http.Error(w, "expired service signature", http.StatusForbidden)
			return
		}
		if _, err := hex.DecodeString(nonce); err != nil {
			http.Error(w, "invalid service signature", http.StatusForbidden)
			return
		}
		providedSignature, err := hex.DecodeString(signature)
		if err != nil {
			http.Error(w, "invalid service signature", http.StatusForbidden)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxInternalBody)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		secret := internalServiceSecret()
		if len(secret) < 32 {
			http.Error(w, "service authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(canonicalServiceRequest(r, body)))
		if !hmac.Equal(providedSignature, mac.Sum(nil)) {
			http.Error(w, "invalid service signature", http.StatusForbidden)
			return
		}

		if db.Rdb == nil {
			http.Error(w, "service authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		namespace := os.Getenv("PIPELINE_NAMESPACE")
		if namespace == "" {
			namespace = "stratai:v2"
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		acquired, err := db.Rdb.SetNX(
			ctx,
			fmt.Sprintf("%s:service-nonce:%s", namespace, nonce),
			"1",
			time.Minute,
		).Result()
		if err != nil {
			http.Error(w, "service authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		if !acquired {
			http.Error(w, "replayed service request", http.StatusConflict)
			return
		}

		next.ServeHTTP(w, r)
	}
}
