package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignatureVerifier_Verify(t *testing.T) {
	secret := "test-secret-key"
	verifier := NewSignatureVerifier(secret)

	tests := []struct {
		name      string
		body      []byte
		signature string
		wantValid bool
	}{
		{
			name:      "valid signature",
			body:      []byte(`{"action":"created"}`),
			signature: computeSignature(secret, []byte(`{"action":"created"}`)),
			wantValid: true,
		},
		{
			name:      "invalid signature",
			body:      []byte(`{"action":"created"}`),
			signature: "invalid-signature",
			wantValid: false,
		},
		{
			name:      "empty signature",
			body:      []byte(`{"action":"created"}`),
			signature: "",
			wantValid: false,
		},
		{
			name:      "tampered body",
			body:      []byte(`{"action":"tampered"}`),
			signature: computeSignature(secret, []byte(`{"action":"created"}`)),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifier.Verify(tt.signature, tt.body)
			if got != tt.wantValid {
				t.Errorf("Verify() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestSignatureVerifier_Middleware(t *testing.T) {
	secret := "test-secret-key"
	verifier := NewSignatureVerifier(secret)

	// Create a test handler that records if it was called
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	tests := []struct {
		name           string
		body           string
		signature      string
		wantStatus     int
		wantHandlerRun bool
	}{
		{
			name:           "valid signature passes",
			body:           `{"action":"created"}`,
			signature:      computeSignature(secret, []byte(`{"action":"created"}`)),
			wantStatus:     http.StatusOK,
			wantHandlerRun: true,
		},
		{
			name:           "invalid signature rejected",
			body:           `{"action":"created"}`,
			signature:      "invalid",
			wantStatus:     http.StatusUnauthorized,
			wantHandlerRun: false,
		},
		{
			name:           "missing signature rejected",
			body:           `{"action":"created"}`,
			signature:      "",
			wantStatus:     http.StatusUnauthorized,
			wantHandlerRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tt.body))
			if tt.signature != "" {
				req.Header.Set("Sentry-Hook-Signature", tt.signature)
			}

			rr := httptest.NewRecorder()
			verifier.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatus)
			}

			if handlerCalled != tt.wantHandlerRun {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantHandlerRun)
			}
		})
	}
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
