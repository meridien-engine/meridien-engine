package middleware

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// buildPayload is a test helper that base64url-encodes a map as the JWT
// payload segment. It intentionally produces no padding (RawURLEncoding) to
// mirror the format expected by parseJWTClaims.
func buildPayload(t *testing.T, claims map[string]string) string {
	t.Helper()
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("buildPayload: failed to marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// fakeJWT assembles a three-segment dot-separated token with a dummy header
// and signature so that parseJWTClaims sees a structurally valid JWT.
func fakeJWT(header, payload, signature string) string {
	return header + "." + payload + "." + signature
}

// dummyHeader is a minimal, non-verified JWT header segment.
const dummyHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// dummySig is a placeholder signature; parseJWTClaims does not verify it.
const dummySig = "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

// ---------------------------------------------------------------------------
// parseJWTClaims tests
// ---------------------------------------------------------------------------

// TestParseJWTClaims_ValidToken verifies that a well-formed JWT with a
// business_id claim is parsed correctly and the claim value is returned.
func TestParseJWTClaims_ValidToken(t *testing.T) {
	const wantBusinessID = "biz-abc-123"

	payload := buildPayload(t, map[string]string{"business_id": wantBusinessID})
	token := fakeJWT(dummyHeader, payload, dummySig)

	got, err := parseJWTClaims(token)
	if err != nil {
		t.Fatalf("parseJWTClaims(%q): unexpected error: %v", token, err)
	}
	if got != wantBusinessID {
		t.Errorf("parseJWTClaims businessID = %q; want %q", got, wantBusinessID)
	}
}

// TestParseJWTClaims_MalformedToken_TooFewParts verifies that a token with
// fewer than three dot-separated segments is rejected with an error.
func TestParseJWTClaims_MalformedToken_TooFewParts(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"one part only", "onlyone"},
		{"two parts", "header.payload"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseJWTClaims(tc.token)
			if err == nil {
				t.Errorf("parseJWTClaims(%q): expected an error for malformed token, got nil", tc.token)
			}
		})
	}
}

// TestParseJWTClaims_MissingBusinessID verifies that a structurally valid JWT
// whose payload does not contain the business_id claim returns an error.
// This guards against accidentally accepting tokens issued by unrelated systems.
func TestParseJWTClaims_MissingBusinessID(t *testing.T) {
	// Payload has a completely different claim; business_id is absent.
	payload := buildPayload(t, map[string]string{"sub": "user-xyz", "email": "test@example.com"})
	token := fakeJWT(dummyHeader, payload, dummySig)

	_, err := parseJWTClaims(token)
	if err == nil {
		t.Errorf("parseJWTClaims(%q): expected error for missing business_id claim, got nil", token)
	}
}

// TestParseJWTClaims_EmptyBusinessID verifies that an explicit empty string
// value for business_id is treated the same as a missing claim.
func TestParseJWTClaims_EmptyBusinessID(t *testing.T) {
	payload := buildPayload(t, map[string]string{"business_id": ""})
	token := fakeJWT(dummyHeader, payload, dummySig)

	_, err := parseJWTClaims(token)
	if err == nil {
		t.Errorf("parseJWTClaims(%q): expected error for empty business_id, got nil", token)
	}
}

// TestParseJWTClaims_InvalidBase64Payload verifies that a payload segment that
// is not valid base64url returns a meaningful error rather than panicking.
func TestParseJWTClaims_InvalidBase64Payload(t *testing.T) {
	token := fakeJWT(dummyHeader, "!!!not-base64!!!", dummySig)

	_, err := parseJWTClaims(token)
	if err == nil {
		t.Errorf("parseJWTClaims(%q): expected error for invalid base64 payload, got nil", token)
	}
}
