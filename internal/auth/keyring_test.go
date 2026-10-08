package auth

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// ── AES-GCM encrypt/decrypt round-trip ──────────────────────────────────────

func TestAESGCM_RoundTrip(t *testing.T) {
	key := deriveMachineKey()
	plaintext := []byte("hello, futu-cli token data")

	encrypted, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data should differ from plaintext")
	}

	decrypted, err := decryptAESGCM(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestAESGCM_DifferentCiphertextEachTime(t *testing.T) {
	key := deriveMachineKey()
	plaintext := []byte("same input")

	enc1, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt 1: %v", err)
	}
	enc2, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt 2: %v", err)
	}

	if bytes.Equal(enc1, enc2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestAESGCM_WrongKey(t *testing.T) {
	key := deriveMachineKey()
	plaintext := []byte("secret")

	encrypted, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	wrongKey := make([]byte, 32)
	copy(wrongKey, key)
	wrongKey[0] ^= 0xFF

	_, err = decryptAESGCM(wrongKey, encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestAESGCM_TruncatedCiphertext(t *testing.T) {
	key := deriveMachineKey()
	_, err := decryptAESGCM(key, []byte("short"))
	if err == nil {
		t.Fatal("expected error for truncated ciphertext")
	}
}

func TestAESGCM_EmptyPlaintext(t *testing.T) {
	key := deriveMachineKey()
	encrypted, err := encryptAESGCM(key, []byte{})
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	decrypted, err := decryptAESGCM(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted, got %d bytes", len(decrypted))
	}
}

// ── EncryptAndSave / DecryptAndLoad round-trip ──────────────────────────────

func TestEncryptAndSave_DecryptAndLoad_RoundTrip(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "token.enc")
	data := []byte(`{"access_token":"at-test","refresh_token":"rt-test"}`)

	if err := EncryptAndSave(path, data); err != nil {
		t.Fatalf("EncryptAndSave: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if bytes.Equal(raw, data) {
		t.Error("file content should be encrypted, not plaintext")
	}

	loaded, err := DecryptAndLoad(path)
	if err != nil {
		t.Fatalf("DecryptAndLoad: %v", err)
	}
	if !bytes.Equal(loaded, data) {
		t.Errorf("loaded = %q, want %q", loaded, data)
	}
}

func TestEncryptAndSaveUsesRandomKeyringKey(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "token.enc")
	data := []byte(`{"access_token":"secret"}`)

	if err := EncryptAndSave(path, data); err != nil {
		t.Fatalf("EncryptAndSave: %v", err)
	}
	encrypted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read encrypted token: %v", err)
	}
	if _, err := decryptAESGCM(deriveMachineKey(), encrypted); err == nil {
		t.Fatal("token should not be decryptable with the legacy machine-derived key")
	}
	key, err := loadEncryptionKey()
	if err != nil {
		t.Fatalf("loadEncryptionKey: %v", err)
	}
	if len(key) != encryptionKeySize {
		t.Fatalf("encryption key length = %d, want %d", len(key), encryptionKeySize)
	}
}

func TestDecryptAndLoadAcceptsLegacyMachineKey(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "legacy-token.enc")
	data := []byte(`{"access_token":"legacy"}`)
	encrypted, err := encryptAESGCM(deriveMachineKey(), data)
	if err != nil {
		t.Fatalf("encrypt legacy token: %v", err)
	}
	if err := os.WriteFile(path, encrypted, tokenFilePermissions); err != nil {
		t.Fatalf("write legacy token: %v", err)
	}

	loaded, err := DecryptAndLoad(path)
	if err != nil {
		t.Fatalf("DecryptAndLoad: %v", err)
	}
	if !bytes.Equal(loaded, data) {
		t.Fatalf("loaded = %q, want %q", loaded, data)
	}
	migrated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migrated token: %v", err)
	}
	if _, err := decryptAESGCM(deriveMachineKey(), migrated); err == nil {
		t.Fatal("legacy token file was not migrated to the keyring-backed key")
	}
}

func TestDecryptAndLoad_FileNotFound(t *testing.T) {
	_, err := DecryptAndLoad("/tmp/futu-cli-test-no-such-file-98765")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

// ── deriveMachineKey ────────────────────────────────────────────────────────

func TestDeriveMachineKey_Deterministic(t *testing.T) {
	k1 := deriveMachineKey()
	k2 := deriveMachineKey()
	if !bytes.Equal(k1, k2) {
		t.Error("deriveMachineKey should return the same key on the same machine")
	}
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32", len(k1))
	}
}

// ── generateNonce ───────────────────────────────────────────────────────────

func TestGenerateNonce_CorrectSize(t *testing.T) {
	nonce, err := generateNonce(12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nonce) != 12 {
		t.Errorf("nonce length = %d, want 12", len(nonce))
	}
}

func TestGenerateNonce_Unique(t *testing.T) {
	n1, _ := generateNonce(12)
	n2, _ := generateNonce(12)
	if bytes.Equal(n1, n2) {
		t.Error("two nonces should not be identical")
	}
}

// ── ensureParentDir ─────────────────────────────────────────────────────────

func TestEnsureParentDir_CreatesNestedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "file.txt")
	if err := ensureParentDir(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parentDir := filepath.Dir(path)
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("parent dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}
