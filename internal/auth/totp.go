package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const totpDigits = 6
const totpStep = 30 // sec

func generateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

func totpNow(secret string) (string, error) {
	return totpAt(secret, time.Now())
}

func totpAt(secret string, t time.Time) (string, error) {
	secret = strings.ToUpper(strings.TrimRight(secret, "="))
	padding := (8 - len(secret)%8) % 8
	secret += strings.Repeat("=", padding)

	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}

	counter := uint64(t.Unix()) / totpStep
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	code := binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff
	code %= uint32(math.Pow10(totpDigits))

	return fmt.Sprintf("%0*d", totpDigits, code), nil
}

func verifyTOTP(secret, code string) bool {
	now := time.Now()
	for _, delta := range []time.Duration{-totpStep * time.Second, 0, totpStep * time.Second} {
		c, err := totpAt(secret, now.Add(delta))
		if err == nil && c == code {
			return true
		}
	}
	return false
}

func totpProvisioningURL(secret, email, issuer string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=%d&period=%d",
		issuer, email, secret, issuer, totpDigits, totpStep,
	)
}
