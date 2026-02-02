package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

// SignatureVerifier verifies Sentry webhook signatures.
type SignatureVerifier struct {
	secret []byte
}

// NewSignatureVerifier creates a new signature verifier.
func NewSignatureVerifier(secret string) *SignatureVerifier {
	return &SignatureVerifier{
		secret: []byte(secret),
	}
}

// Verify checks the HMAC-SHA256 signature of the request body.
// The signature is expected in the Sentry-Hook-Signature header.
func (v *SignatureVerifier) Verify(signature string, body []byte) bool {
	if signature == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	// Compare using constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expected))
}

// Middleware returns an HTTP middleware that verifies webhook signatures.
func (v *SignatureVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read signature from header
		signature := r.Header.Get("Sentry-Hook-Signature")
		if signature == "" {
			http.Error(w, "missing signature", http.StatusUnauthorized)
			return
		}

		// Read and buffer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Verify signature
		if !v.Verify(signature, body) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		// Restore body for downstream handlers
		r.Body = io.NopCloser(newBodyReader(body))

		next.ServeHTTP(w, r)
	})
}

// bodyReader implements io.Reader for buffered body.
type bodyReader struct {
	data []byte
	pos  int
}

func newBodyReader(data []byte) *bodyReader {
	return &bodyReader{data: data}
}

func (b *bodyReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
