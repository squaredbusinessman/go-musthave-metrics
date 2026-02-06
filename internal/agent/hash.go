package agent

import (
	"crypto/sha256"
	"encoding/hex"
)

func sha256hex(body []byte, key string) string {
	sum := sha256.Sum256(append(body, []byte(key)...))
	return hex.EncodeToString(sum[:])
}
