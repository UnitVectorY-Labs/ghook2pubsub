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
	"net/url"
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
	req.Header.Set("User-Agent", "GitHub-Hookshot/test")
	req.Header.Set("X-Hub-Signature-256", computeSignature(body, secret))
	req.Header.Set("X-GitHub-Delivery", "delivery-id")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Hook-ID", "hook-id")
	req.Header.Set("X-GitHub-Hook-Installation-Target-Type", "organization")
	req.Header.Set("X-GitHub-Hook-Installation-Target-ID", "123")
	req.Header.Set("Content-Type", "application/json")
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

func TestHandler_InvalidUserAgent(t *testing.T) {
	pub := &mockPublisher{
		publishFunc: func(ctx context.Context, data []byte, attrs map[string]string) (string, error) {
			t.Fatal("publisher should not be called")
			return "", nil
		},
	}

	metrics := &Metrics{}
	handler := NewHandler(pub, []string{"secret"}, metrics)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("User-Agent", "curl/8.0.0")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if metrics.UserAgentFailures.Load() != 1 {
		t.Errorf("UserAgentFailures = %d, want 1", metrics.UserAgentFailures.Load())
	}
	if metrics.SignatureFailures.Load() != 0 {
		t.Errorf("SignatureFailures = %d, want 0", metrics.SignatureFailures.Load())
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
	req.Header.Set("User-Agent", "GitHub-Hookshot/test")
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
	req.Header.Set("User-Agent", "GitHub-Hookshot/test")
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
	if receivedAttrs["gh_delivery"] != "delivery-id" {
		t.Errorf("gh_delivery = %q, want %q", receivedAttrs["gh_delivery"], "delivery-id")
	}
	if receivedAttrs["gh_event"] != "push" {
		t.Errorf("gh_event = %q, want %q", receivedAttrs["gh_event"], "push")
	}
	if receivedAttrs["gh_hook_id"] != "hook-id" {
		t.Errorf("gh_hook_id = %q, want %q", receivedAttrs["gh_hook_id"], "hook-id")
	}
	if receivedAttrs["gh_target_type"] != "organization" {
		t.Errorf("gh_target_type = %q, want %q", receivedAttrs["gh_target_type"], "organization")
	}
	if receivedAttrs["gh_target_id"] != "123" {
		t.Errorf("gh_target_id = %q, want %q", receivedAttrs["gh_target_id"], "123")
	}
	if receivedAttrs["action"] != "opened" {
		t.Errorf("action = %q, want %q", receivedAttrs["action"], "opened")
	}
	if receivedAttrs["organization"] != "org" {
		t.Errorf("organization = %q, want %q", receivedAttrs["organization"], "org")
	}
	if receivedAttrs["repository"] != "org/repo" {
		t.Errorf("repository = %q, want %q", receivedAttrs["repository"], "org/repo")
	}
}

func TestHandler_FormEncodedPayload(t *testing.T) {
	secret := "test-secret"
	body := []byte("payload=" + url.QueryEscape(`{"repository":{"full_name":"org/repo"},"sender":{"login":"octocat"}}`))

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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if receivedAttrs["repository"] != "org/repo" {
		t.Errorf("repository = %q, want %q", receivedAttrs["repository"], "org/repo")
	}
	if receivedAttrs["sender"] != "octocat" {
		t.Errorf("sender = %q, want %q", receivedAttrs["sender"], "octocat")
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
	if metrics.UserAgentFailures.Load() != 0 {
		t.Errorf("UserAgentFailures = %d, want 0", metrics.UserAgentFailures.Load())
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

	var payload map[string]string
	if err := json.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if payload["status"] != "ok" {
		t.Errorf("status field = %q, want %q", payload["status"], "ok")
	}
}
