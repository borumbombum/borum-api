package auth

import (
	"crypto/sha256"
	"encoding/base64"
)

// base64URL encodes bytes without padding so tokens are safe in URLs and
// cookies.
func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// hashToken returns the SHA-256 digest of a session token. Only this digest
// is stored in the database, so a leaked table does not leak live sessions.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64URL(h[:])
}
