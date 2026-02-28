package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature checks the HMAC-SHA256 signature against multiple secrets.
// Returns true if any secret produces a matching signature.
func VerifySignature(body []byte, signatureHeader string, secrets []string) bool {
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	sigHex := strings.TrimPrefix(signatureHeader, "sha256=")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	for _, secret := range secrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := mac.Sum(nil)
		if hmac.Equal(sigBytes, expected) {
			return true
		}
	}

	return false
}
