package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func VerifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := CalculateChecksum(data)
	return actualChecksum == expectedChecksum
}