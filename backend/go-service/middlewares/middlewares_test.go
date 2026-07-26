package middlewares

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

type serviceAuthVector struct {
	Version      string `json:"version"`
	Method       string `json:"method"`
	PathAndQuery string `json:"path_and_query"`
	Timestamp    string `json:"timestamp"`
	Nonce        string `json:"nonce"`
	Body         string `json:"body"`
	Secret       string `json:"secret"`
	Signature    string `json:"signature"`
}

func TestCanonicalServiceRequestMatchesSharedVector(t *testing.T) {
	data, err := os.ReadFile("../../testdata/service_auth_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector serviceAuthVector
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(vector.Method, vector.PathAndQuery, nil)
	request.Header.Set("X-Service-Version", vector.Version)
	request.Header.Set("X-Service-Timestamp", vector.Timestamp)
	request.Header.Set("X-Service-Nonce", vector.Nonce)
	canonical := canonicalServiceRequest(request, []byte(vector.Body))

	mac := hmac.New(sha256.New, []byte(vector.Secret))
	_, _ = mac.Write([]byte(canonical))
	if hex.EncodeToString(mac.Sum(nil)) != vector.Signature {
		t.Fatalf("signature does not match shared vector")
	}

	tampered := httptest.NewRequest(vector.Method, "/process-demo?x=2", nil)
	tampered.Header = request.Header.Clone()
	mac.Reset()
	_, _ = mac.Write([]byte(canonicalServiceRequest(tampered, []byte(vector.Body))))
	if hex.EncodeToString(mac.Sum(nil)) == vector.Signature {
		t.Fatalf("tampered path unexpectedly matched signature")
	}
}

func TestInternalServiceSecretMatchesDevelopmentDerivation(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("INTERNAL_SERVICE_SECRET", "")
	t.Setenv("SESSION_SECRET_KEY", "development-session-secret")

	expected := sha256.Sum256([]byte("development-session-secret:internal-service"))
	if internalServiceSecret() != hex.EncodeToString(expected[:]) {
		t.Fatal("development service secret does not match Node derivation")
	}
}

func TestInternalServiceSecretRequiresExplicitProductionValue(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("INTERNAL_SERVICE_SECRET", "")
	t.Setenv("SESSION_SECRET_KEY", "production-session-secret")

	if internalServiceSecret() != "" {
		t.Fatal("production unexpectedly derived an internal service secret")
	}
}
