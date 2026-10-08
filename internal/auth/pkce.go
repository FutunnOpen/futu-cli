package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	codeVerifierBytes   = 32
	codeChallengeMethod = "S256"
)

// PKCEParams holds the code_verifier and code_challenge for a single
// OAuth authorization request (RFC 7636).
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string
}

// GeneratePKCE creates a fresh PKCE code_verifier and its S256 code_challenge.
func GeneratePKCE() (*PKCEParams, error) {
	verifier, err := randomBase64URL(codeVerifierBytes)
	if err != nil {
		return nil, fmt.Errorf("generate code verifier: %w", err)
	}
	challenge := computeS256Challenge(verifier)
	return &PKCEParams{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		Method:        codeChallengeMethod,
	}, nil
}

// computeS256Challenge returns base64url_no_padding(sha256(verifier)).
func computeS256Challenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// randomBase64URL generates n random bytes and returns them as a
// base64url-encoded string without padding.
func randomBase64URL(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
