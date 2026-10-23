package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksumSignatureAcceptsEmbeddedKey(t *testing.T) {
	checksums := []byte("abc123  futu_v0.1.0_darwin_arm64.tar.gz\n")
	key := mustLocalReleaseKey(t)
	signature := signWithKey(t, checksums, key)
	if err := verifyChecksumSignature(checksums, signature); err != nil {
		t.Fatalf("verifyChecksumSignature() error = %v", err)
	}
}

func TestVerifyChecksumSignatureRejectsTamperedChecksums(t *testing.T) {
	checksums := []byte("abc123  futu_v0.1.0_darwin_arm64.tar.gz\n")
	key := mustLocalReleaseKey(t)
	signature := signWithKey(t, checksums, key)
	tampered := append([]byte{}, checksums...)
	tampered[0] = 'z'
	if err := verifyChecksumSignature(tampered, signature); err == nil {
		t.Fatal("expected signature verification to fail for tampered checksums")
	}
}

func TestVerifyPackageVersionReadsArchiveMetadata(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.tar.gz")
	writeTestTarGzip(t, archive, map[string]string{
		versionFileName: "v1.2.3\n",
		binaryName:      "unused",
	})
	if err := verifyPackageVersion(archive, archiveExtTarGzip, "v1.2.3"); err != nil {
		t.Fatalf("verifyPackageVersion() error = %v", err)
	}
	if err := verifyPackageVersion(archive, archiveExtTarGzip, "v9.9.9"); err == nil {
		t.Fatal("expected version mismatch")
	}
}

func mustLocalReleaseKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "keys", "release.key.local"),
		"/tmp/futu-cli-release.key",
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		block, _ := pem.Decode(data)
		if block == nil {
			continue
		}
		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return key
		}
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			continue
		}
		key, ok := parsed.(*ecdsa.PrivateKey)
		if ok {
			return key
		}
	}
	t.Skip("local release signing key not available")
	return nil
}

func signWithKey(t *testing.T, payload []byte, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	hash := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signature
}

func writeTestTarGzip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write content: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
}
