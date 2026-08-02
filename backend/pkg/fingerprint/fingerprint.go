package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/Nesio-app/Nesio_go/internal/models"
)

func Fingerprint(signal models.Signal) string {
	// Canonicalize fields
	keys := make([]string, 0, len(signal.Fields))
	for k := range signal.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonical string
	for _, k := range keys {
		canonical += fmt.Sprintf("%s=%v;", k, signal.Fields[k])
	}

	content := fmt.Sprintf("v2:%s:%s:%s", signal.Source, signal.AnchorID, canonical)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8]) // 16 hex chars
}
