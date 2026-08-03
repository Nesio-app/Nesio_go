package vision

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PHashBytes returns a compact deterministic hash used as visual fallback fingerprint.
func PHashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}

// HammingDistanceHex returns a bit distance between two hex-encoded hashes.
func HammingDistanceHex(a, b string) int {
	a = strings.TrimSpace(strings.ToLower(a))
	b = strings.TrimSpace(strings.ToLower(b))
	if a == "" || b == "" || len(a) != len(b) {
		return 64
	}
	distance := 0
	for i := 0; i < len(a); i++ {
		xor := hexNibble(a[i]) ^ hexNibble(b[i])
		distance += bitsSet(xor)
	}
	return distance
}

func hexNibble(v byte) uint8 {
	switch {
	case v >= '0' && v <= '9':
		return v - '0'
	case v >= 'a' && v <= 'f':
		return v - 'a' + 10
	default:
		return 0
	}
}

func bitsSet(v uint8) int {
	count := 0
	for v > 0 {
		count += int(v & 1)
		v >>= 1
	}
	return count
}
