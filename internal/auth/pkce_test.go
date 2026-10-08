package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGeneratePKCE_Fields(t *testing.T) {
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Method != codeChallengeMethod {
		t.Errorf("Method = %q, want %q", p.Method, codeChallengeMethod)
	}
	if len(p.CodeVerifier) == 0 {
		t.Error("CodeVerifier should not be empty")
	}
	if len(p.CodeChallenge) == 0 {
		t.Error("CodeChallenge should not be empty")
	}
}

func TestGeneratePKCE_VerifierIsBase64URL(t *testing.T) {
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(p.CodeVerifier, "+/=") {
		t.Errorf("verifier %q contains non-base64url characters", p.CodeVerifier)
	}
	if _, err := base64.RawURLEncoding.DecodeString(p.CodeVerifier); err != nil {
		t.Errorf("verifier is not valid base64url: %v", err)
	}
}

func TestGeneratePKCE_ChallengeIsBase64URL(t *testing.T) {
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(p.CodeChallenge, "+/=") {
		t.Errorf("challenge %q contains non-base64url characters", p.CodeChallenge)
	}
	if _, err := base64.RawURLEncoding.DecodeString(p.CodeChallenge); err != nil {
		t.Errorf("challenge is not valid base64url: %v", err)
	}
}

func TestGeneratePKCE_ChallengeMatchesVerifier(t *testing.T) {
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	expected := computeS256Challenge(p.CodeVerifier)
	if p.CodeChallenge != expected {
		t.Errorf("challenge = %q, want %q (SHA256 of verifier)", p.CodeChallenge, expected)
	}
}

func TestGeneratePKCE_Unique(t *testing.T) {
	p1, _ := GeneratePKCE()
	p2, _ := GeneratePKCE()
	if p1.CodeVerifier == p2.CodeVerifier {
		t.Error("two PKCE pairs should have different verifiers")
	}
}

func TestComputeS256Challenge_KnownVector(t *testing.T) {
	// RFC 7636 Appendix B example (adapted):
	// verifier "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk" →
	// challenge "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	got := computeS256Challenge(verifier)
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got != want {
		t.Errorf("S256(%q) = %q, want %q", verifier, got, want)
	}
}
