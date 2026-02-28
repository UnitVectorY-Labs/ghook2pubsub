package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/publisher"
)

// Ensure mockPublisher implements publisher.Publisher at compile time.
var _ publisher.Publisher = (*mockPublisher)(nil)

type mockPublisher struct {
	publishFunc func(ctx context.Context, data []byte, attrs map[string]string) (string, error)
}

func (m *mockPublisher) Publish(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
	return m.publishFunc(ctx, data, attrs)
}

func (m *mockPublisher) Close() error { return nil }

func computeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newWebhookRequest(body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", computeSignature(body, secret))
	req.Header.Set("X-GitHub-Delivery", "delivery-id")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Hook-ID", "hook-id")
	return req
}

func TestHandler_SuccessfulDelivery(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened","repository":{"full_name":"org/repo"},"organization":{"login":"org"}}`)
	published := false

	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			published = true
			return "msg-123", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{secret}, metrics)

	req := newWebhookRequest(body, secret)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !published {
		t.Error("expected publisher to be called")
	}
}

func TestHandler_MissingSignature(t *testing.T) {
	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			t.Fatal("publisher should not be called")
			return "", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{"secret"}, metrics)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	// No X-Hub-Signature-256 header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if metrics.SignatureFailures.Load() != 1 {
		t.Errorf("SignatureFailures = %d, want 1", metrics.SignatureFailures.Load())
	}
}

func TestHandler_InvalidSignature(t *testing.T) {
	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			t.Fatal("publisher should not be called")
			return "", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{"correct-secret"}, metrics)

	body := []byte(`{"action":"opened"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", computeSignature(body, "wrong-secret"))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if metrics.SignatureFailures.Load() != 1 {
		t.Errorf("SignatureFailures = %d, want 1", metrics.SignatureFailures.Load())
	}
}

func TestHandler_PublisherFailure(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened"}`)

	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			return "", errors.New("pubsub error")
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{secret}, metrics)

	req := newWebhookRequest(body, secret)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if metrics.PublishFailures.Load() != 1 {
		t.Errorf("PublishFailures = %d, want 1", metrics.PublishFailures.Load())
	}
}

func TestHandler_RawBodyPassedToPublisher(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened","repository":{"full_name":"o/r"},"organization":{"login":"o"}}`)

	var receivedData []byte
	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			receivedData = data
			return "id", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{secret}, metrics)

	req := newWebhookRequest(body, secret)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !bytes.Equal(receivedData, body) {
		t.Errorf("published data does not match original body")
	}
}

func TestHandler_ExtractedAttributesPassedToPublisher(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened","repository":{"full_name":"org/repo"},"organization":{"login":"org"}}`)

	var receivedAttrs map[string]string
	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			receivedAttrs = attrs
			return "id", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{secret}, metrics)

	req := newWebhookRequest(body, secret)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if receivedAttrs["delivery"] != "delivery-id" {
		t.Errorf("delivery = %q, want %q", receivedAttrs["delivery"], "delivery-id")
	}
	if receivedAttrs["gh_event"] != "push" {
		t.Errorf("gh_event = %q, want %q", receivedAttrs["gh_event"], "push")
	}
	if receivedAttrs["action"] != "opened" {
		t.Errorf("action = %q, want %q", receivedAttrs["action"], "opened")
	}
	if receivedAttrs["org"] != "org" {
		t.Errorf("org = %q, want %q", receivedAttrs["org"], "org")
	}
	if receivedAttrs["repo"] != "org/repo" {
		t.Errorf("repo = %q, want %q", receivedAttrs["repo"], "org/repo")
	}
}

func TestHandler_MetricsIncremented(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened","repository":{"full_name":"o/r"},"organization":{"login":"o"}}`)

	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			return "id", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{secret}, metrics)

	req := newWebhookRequest(body, secret)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if metrics.TotalRequests.Load() != 1 {
		t.Errorf("TotalRequests = %d, want 1", metrics.TotalRequests.Load())
	}
	if metrics.PublishSuccesses.Load() != 1 {
		t.Errorf("PublishSuccesses = %d, want 1", metrics.PublishSuccesses.Load())
	}
	if metrics.SignatureFailures.Load() != 0 {
		t.Errorf("SignatureFailures = %d, want 0", metrics.SignatureFailures.Load())
	}
	if metrics.PublishFailures.Load() != 0 {
		t.Errorf("PublishFailures = %d, want 0", metrics.PublishFailures.Load())
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	respBody, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %q, want %q", result["status"], "ok")
	}
}
