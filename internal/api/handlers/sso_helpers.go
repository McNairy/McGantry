package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func hashEmailForLog(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:8])
}

func writeSSOProviderError(w http.ResponseWriter, provider, operation string, err error) {
	log.Printf("%s auth: %s: %v", provider, operation, err)
	writeError(w, http.StatusBadGateway, fmt.Sprintf("%s authentication failed", provider))
}

// randomHex16 generates 16 random bytes as a hex string.
func randomHex16() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
