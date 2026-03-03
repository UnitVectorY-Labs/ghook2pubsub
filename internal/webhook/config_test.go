package webhook

import (
	"os"
	"testing"

	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/publisher"
)

func clearConfigEnv() {
	os.Unsetenv("PUBSUB_PROJECT_ID")
	os.Unsetenv("PUBSUB_TOPIC_ID")
	os.Unsetenv("WEBHOOK_SECRETS")
	os.Unsetenv("LISTEN_ADDR")
	os.Unsetenv("LISTEN_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("PAYLOAD_COMPRESSION")
	os.Unsetenv("PAYLOAD_COMPRESSION_ATTRIBUTE")
}

func TestLoadConfig_Valid(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PubSubProjectID != "my-project" {
		t.Errorf("PubSubProjectID = %q, want %q", cfg.PubSubProjectID, "my-project")
	}
	if cfg.PubSubTopicID != "my-topic" {
		t.Errorf("PubSubTopicID = %q, want %q", cfg.PubSubTopicID, "my-topic")
	}
	if len(cfg.WebhookSecrets) != 1 || cfg.WebhookSecrets[0] != "secret1" {
		t.Errorf("WebhookSecrets = %v, want [secret1]", cfg.WebhookSecrets)
	}
	if cfg.PayloadCompression.Enabled() {
		t.Errorf("PayloadCompression.Enabled() = true, want false")
	}
}

func TestLoadConfig_MissingProjectID(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing PUBSUB_PROJECT_ID")
	}
}

func TestLoadConfig_MissingTopicID(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("WEBHOOK_SECRETS", "secret1")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing PUBSUB_TOPIC_ID")
	}
}

func TestLoadConfig_MissingSecrets(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing WEBHOOK_SECRETS")
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "0.0.0.0")
	}
	if cfg.ListenPort != "8080" {
		t.Errorf("ListenPort = %q, want %q", cfg.ListenPort, "8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.PayloadCompression.String() != "none" {
		t.Errorf("PayloadCompression = %q, want %q", cfg.PayloadCompression.String(), "none")
	}
	if cfg.PayloadCompression.AttributeName != "" {
		t.Errorf("PayloadCompression.AttributeName = %q, want empty", cfg.PayloadCompression.AttributeName)
	}
}

func TestLoadConfig_MultipleSecrets(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1,secret2,secret3")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.WebhookSecrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(cfg.WebhookSecrets))
	}
	expected := []string{"secret1", "secret2", "secret3"}
	for i, want := range expected {
		if cfg.WebhookSecrets[i] != want {
			t.Errorf("WebhookSecrets[%d] = %q, want %q", i, cfg.WebhookSecrets[i], want)
		}
	}
}

func TestLoadConfig_PayloadCompression(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1")
	t.Setenv("PAYLOAD_COMPRESSION", "gzip:6")
	t.Setenv("PAYLOAD_COMPRESSION_ATTRIBUTE", "compression")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PayloadCompression.Algorithm != publisher.CompressionGzip {
		t.Fatalf("PayloadCompression.Algorithm = %q, want %q", cfg.PayloadCompression.Algorithm, publisher.CompressionGzip)
	}
	if cfg.PayloadCompression.Level != 6 {
		t.Fatalf("PayloadCompression.Level = %d, want 6", cfg.PayloadCompression.Level)
	}
	if cfg.PayloadCompression.AttributeName != "compression" {
		t.Fatalf("PayloadCompression.AttributeName = %q, want %q", cfg.PayloadCompression.AttributeName, "compression")
	}
}

func TestLoadConfig_InvalidPayloadCompression(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", "secret1")
	t.Setenv("PAYLOAD_COMPRESSION", "gzip")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid PAYLOAD_COMPRESSION")
	}
}

func TestLoadConfig_SecretsWhitespaceTrimming(t *testing.T) {
	clearConfigEnv()
	t.Setenv("PUBSUB_PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_TOPIC_ID", "my-topic")
	t.Setenv("WEBHOOK_SECRETS", " secret1 , secret2 , secret3 ")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.WebhookSecrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(cfg.WebhookSecrets))
	}
	expected := []string{"secret1", "secret2", "secret3"}
	for i, want := range expected {
		if cfg.WebhookSecrets[i] != want {
			t.Errorf("WebhookSecrets[%d] = %q, want %q", i, cfg.WebhookSecrets[i], want)
		}
	}
}
