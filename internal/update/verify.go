package update

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
)

const checksumSignatureSuffix = ".sig"

//go:embed release.pub
var releasePublicKeyPEM []byte

func verifyChecksumSignature(checksums, signature []byte) error {
	pub, err := parseReleasePublicKey(releasePublicKeyPEM)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(checksums)
	if !ecdsa.VerifyASN1(pub, hash[:], signature) {
		return errors.New("checksum signature verification failed")
	}
	return nil
}

func parseReleasePublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("decode release public key: no PEM block")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse release public key: %w", err)
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("release public key is not ECDSA")
	}
	return pub, nil
}
