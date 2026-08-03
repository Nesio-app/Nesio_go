package vision

import (
	"crypto/sha256"
	"encoding/hex"
)

// PHashBytes returns a compact deterministic hash used as visual fallback fingerprint.
func PHashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}
