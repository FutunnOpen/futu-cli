package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	tokenFilePermissions  = 0o600
	tokenDirPermissions   = 0o700
	temporaryTokenPattern = ".futu-token-*"
	keyringService        = "futu-cli"
	keyringUser           = "default"
	encryptionKeyUser     = "token-encryption-key"
	encryptionKeySize     = 32
)

// LoadFromKeyring reads a token created by older CLI versions.
func LoadFromKeyring() ([]byte, error) {
	secret, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

// DeleteFromKeyring removes a token created by older CLI versions.
func DeleteFromKeyring() error {
	return keyring.Delete(keyringService, keyringUser)
}

// EncryptAndSave saves AES-GCM encrypted data using a random key protected by
// the operating system keyring.
func EncryptAndSave(path string, data []byte) error {
	key, err := getOrCreateEncryptionKey()
	if err != nil {
		return fmt.Errorf("load token encryption key: %w", err)
	}
	encrypted, err := encryptAESGCM(key, data)
	if err != nil {
		return fmt.Errorf("encrypt token: %w", err)
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return writeFileAtomically(path, encrypted)
}

func writeFileAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, temporaryTokenPattern)
	if err != nil {
		return fmt.Errorf("create temporary token file: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(tokenFilePermissions); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set token file permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write token file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close token file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace token file: %w", err)
	}
	return nil
}

// DecryptAndLoad loads and decrypts AES-GCM encrypted data from the specified file path.
func DecryptAndLoad(path string) ([]byte, error) {
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}
	key, keyErr := loadEncryptionKey()
	if keyErr == nil {
		if decrypted, decryptErr := decryptAESGCM(key, encrypted); decryptErr == nil {
			return decrypted, nil
		}
	}
	decrypted, legacyErr := decryptAESGCM(deriveMachineKey(), encrypted)
	if legacyErr != nil {
		if keyErr != nil {
			return nil, fmt.Errorf("load token encryption key: %w", keyErr)
		}
		return nil, legacyErr
	}
	if err := EncryptAndSave(path, decrypted); err != nil {
		return nil, fmt.Errorf("migrate legacy token encryption: %w", err)
	}
	return decrypted, nil
}

func getOrCreateEncryptionKey() ([]byte, error) {
	key, err := loadEncryptionKey()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return nil, err
	}
	key = make([]byte, encryptionKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := keyring.Set(keyringService, encryptionKeyUser, encoded); err != nil {
		return nil, fmt.Errorf("store encryption key: %w", err)
	}
	return key, nil
}

func loadEncryptionKey() ([]byte, error) {
	encoded, err := keyring.Get(keyringService, encryptionKeyUser)
	if err != nil {
		return nil, err
	}
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != encryptionKeySize {
		return nil, fmt.Errorf("invalid encryption key length: %d", len(key))
	}
	return key, nil
}

// deriveMachineKey produces a 32-byte AES key from hostname + username.
func deriveMachineKey() []byte {
	hostname, _ := os.Hostname()
	u, _ := user.Current()
	username := ""
	if u != nil {
		username = u.Username
	}
	hash := sha256.Sum256([]byte(hostname + ":" + username))
	return hash[:]
}

// encryptAESGCM encrypts plaintext using AES-256-GCM with a random nonce.
// The returned ciphertext is nonce || encrypted_data.
func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	nonce, err := generateNonce(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptAESGCM decrypts ciphertext that was produced by encryptAESGCM.
func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	encrypted := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, encrypted, nil)
}

// generateNonce produces a cryptographically random nonce of the given size.
func generateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

// ensureParentDir creates the parent directory of path if it does not exist.
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, tokenDirPermissions)
}
