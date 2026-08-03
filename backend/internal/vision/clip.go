package vision

import (
	"crypto/sha256"
	"encoding/binary"
)

// EmbeddingFromBytes produces a deterministic 16-dim embedding for fallback matching.
// This is a lightweight stand-in when external embedding services are unavailable.
func EmbeddingFromBytes(content []byte) []float32 {
	sum := sha256.Sum256(content)
	vec := make([]float32, 16)
	for i := 0; i < 16; i++ {
		chunk := binary.BigEndian.Uint16(sum[i*2 : i*2+2])
		vec[i] = float32(chunk) / 65535.0
	}
	return vec
}
