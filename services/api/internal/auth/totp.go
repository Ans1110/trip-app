package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strconv"
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

func verifyTOTP(secret, code string) (int64, bool) {
	now := time.Now()
	codeBytes := []byte(code)
	var matched int64 = -1
	for _, delta := range []time.Duration{-totpStep * time.Second, 0, totpStep * time.Second} {
		t := now.Add(delta)
		c, err := totpAt(secret, t)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(c), codeBytes) == 1 {
			matched = int64(t.Unix()) / totpStep
		}
	}
	return matched, matched >= 0
}

func totpProvisioningURL(secret, account, issuer string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", strconv.Itoa(totpDigits))
	q.Set("period", strconv.Itoa(totpStep))
	return "otpauth://totp/" + label + "?" + q.Encode()
}
