package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func computeTestSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_ValidSingleSecret(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := "mysecret"
	sig := computeTestSignature(body, secret)

	if !VerifySignature(body, sig, []string{secret}) {
		t.Fatal("expected valid signature to pass")
	}
}

func TestVerifySignature_ValidMultipleSecrets(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := "second-secret"
	sig := computeTestSignature(body, secret)

	if !VerifySignature(body, sig, []string{"first-secret", "second-secret", "third-secret"}) {
		t.Fatal("expected signature matching second secret to pass")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	sig := computeTestSignature(body, "wrong-secret")

	if VerifySignature(body, sig, []string{"correct-secret"}) {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestVerifySignature_MissingPrefix(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	sigNoPrefix := hex.EncodeToString(mac.Sum(nil))

	if VerifySignature(body, sigNoPrefix, []string{"secret"}) {
		t.Fatal("expected signature without sha256= prefix to fail")
	}
}

func TestVerifySignature_EmptyHeader(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	if VerifySignature(body, "", []string{"secret"}) {
		t.Fatal("expected empty signature header to fail")
	}
}

func TestVerifySignature_InvalidHex(t *testing.T) {
	if VerifySignature([]byte(`{}`), "sha256=notvalidhex!!!", []string{"secret"}) {
		t.Fatal("expected invalid hex to fail")
	}
}

func TestVerifySignature_EmptySecretsList(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	sig := computeTestSignature(body, "secret")

	if VerifySignature(body, sig, []string{}) {
		t.Fatal("expected empty secrets list to fail")
	}
}
