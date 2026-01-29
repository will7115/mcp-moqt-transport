package mcpmoqt

import (
	"crypto/rand"
	"encoding/hex"
)

func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

