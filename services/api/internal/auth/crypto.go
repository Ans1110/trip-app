package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var ErrInvalidEncryptionKey = errors.New("encryption key must be 32 bytes (64 hex chars)")
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

func newGCMCipher(keyHex string) (cipher.AEAD, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, ErrInvalidEncryptionKey
	}
	if len(key) != 32 {
		return nil, ErrInvalidEncryptionKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptSecret(plain, keyHex string) (string, error) {
	g, err := newGCMCipher(keyHex)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := g.Seal(nonce, nonce, []byte(plain), nil)
	return hex.EncodeToString(ct), nil
}

func decryptSecret(encryptedHex, keyHex string) (string, error) {
	g, err := newGCMCipher(keyHex)
	if err != nil {
		return "", err
	}
	ct, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	if len(ct) < g.NonceSize() {
		return "", ErrInvalidCiphertext
	}
	nonce := ct[:g.NonceSize()]
	pt, err := g.Open(nil, nonce, ct[g.NonceSize():], nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(pt), nil
}

// deviceFingerprint returns a stable 64-char hex digest of the device signals
// the caller provides. Empty inputs are tolerated; the result is "" only when
// every signal is empty.
func deviceFingerprint(d DeviceInfo) string {
	if d.UserAgent == "" && d.IPAddress == "" && d.DeviceName == "" && d.DeviceType == "" {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(d.UserAgent))
	h.Write([]byte{0})
	h.Write([]byte(d.IPAddress))
	h.Write([]byte{0})
	h.Write([]byte(d.DeviceName))
	h.Write([]byte{0})
	h.Write([]byte(d.DeviceType))
	return hex.EncodeToString(h.Sum(nil))
}
