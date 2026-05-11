package auth

import "gorm.io/gorm"

// RepositoryForTest is a type alias that lets external test packages
// type-assert to the concrete repository and access the underlying DB
// for verification queries (e.g. confirming soft-deletes wrote revoked_at).
// Only compiled during tests.
type RepositoryForTest = repository

func (r *repository) DB() *gorm.DB {
	return r.db
}

var (
	HashToken         = hashToken
	TotpNow           = totpNow
	EncryptSecret     = encryptSecret
	DecryptSecret     = decryptSecret
	DeviceFingerprint = deviceFingerprint
)
