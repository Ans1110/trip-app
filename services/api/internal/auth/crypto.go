package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

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
