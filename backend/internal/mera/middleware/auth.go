// Package middleware provides HTTP and gRPC authentication middleware for the
// Meridien Engine backend. It enforces per-request tenant isolation by
// extracting a business ID from incoming credentials and injecting it into the
// request context via the repository tenant helpers.
package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/meridien-engine/meridien-engine/internal/repository"
)

// jwtPayload holds the subset of JWT claims that the middleware cares about.
// Additional standard claims (iss, exp, iat, …) are intentionally ignored at
// this layer — they must be validated once a proper signing-key verification
// step is wired in (see TODO below).
type jwtPayload struct {
	BusinessID string `json:"business_id"`
}

// parseJWTClaims extracts the business_id claim from a raw JWT string.
//
// The function only decodes the payload section; it does NOT verify the
// signature. This is intentional for the initial bootstrap phase where the
// signing key has not yet been provisioned into the service configuration.
//
// TODO(security): Wire real HMAC-SHA256 / RS256 signature verification here
// before promoting to production. Use the configured signing key / public key
// to validate the third segment (signature) against header + payload.
// Accepting unsigned or arbitrarily signed tokens is a critical security risk.
func parseJWTClaims(token string) (businessID string, err error) {
	// Remove any accidental whitespace (e.g., from copy-pasting)
	token = strings.ReplaceAll(token, " ", "")
	
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("malformed JWT: expected 3 dot-separated segments, got %d", len(parts))
	}

	// The payload is the middle segment, Base64URL-encoded without padding.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("malformed JWT payload: base64 decode failed: %w", err)
	}

	var claims jwtPayload
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", fmt.Errorf("malformed JWT payload: JSON unmarshal failed: %w", err)
	}

	if claims.BusinessID == "" {
		return "", errors.New("JWT payload missing required claim: business_id")
	}

	return claims.BusinessID, nil
}

// JWTAuth is an HTTP middleware that enforces JWT-based tenant authentication.
//
// It reads the "Authorization: Bearer <token>" header, calls parseJWTClaims to
// extract the business_id, and stores it in the request context via
// repository.WithBusinessID. Downstream handlers can retrieve it with
// repository.BusinessIDFromContext.
//
// Requests without a valid Authorization header or with an unparseable token
// are rejected with HTTP 401 Unauthorized. No 403 is returned at this layer —
// authorisation decisions belong to individual handlers.
//
// Usage:
//
//	r.Use(middleware.JWTAuth)
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass CORS preflight requests through
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expect exactly "Bearer <token>" — anything else is rejected.
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			http.Error(w, "Authorization header must use Bearer scheme", http.StatusUnauthorized)
			return
		}

		rawToken := strings.TrimPrefix(authHeader, bearerPrefix)
		if rawToken == "" {
			http.Error(w, "Bearer token is empty", http.StatusUnauthorized)
			return
		}

		businessID, err := parseJWTClaims(rawToken)
		if err != nil {
			http.Error(w, "invalid or unparseable JWT: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := repository.WithBusinessID(r.Context(), businessID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
