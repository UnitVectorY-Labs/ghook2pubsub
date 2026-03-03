package webhook

import (
	"fmt"
	"os"
	"strings"

	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/publisher"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	PubSubProjectID    string
	PubSubTopicID      string
	WebhookSecrets     []string
	ListenAddr         string
	ListenPort         string
	LogLevel           string
	PayloadCompression publisher.CompressionConfig
}

// LoadConfig reads configuration from environment variables and validates required fields.
func LoadConfig() (*Config, error) {
	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("PUBSUB_PROJECT_ID is required")
	}

	topicID := os.Getenv("PUBSUB_TOPIC_ID")
	if topicID == "" {
		return nil, fmt.Errorf("PUBSUB_TOPIC_ID is required")
	}

	secretsRaw := os.Getenv("WEBHOOK_SECRETS")
	if secretsRaw == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRETS is required")
	}

	var secrets []string
	for _, s := range strings.Split(secretsRaw, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			secrets = append(secrets, trimmed)
		}
	}
	if len(secrets) == 0 {
		return nil, fmt.Errorf("WEBHOOK_SECRETS must contain at least one non-empty secret")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0"
	}

	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	payloadCompression, err := publisher.ParseCompressionConfig(
		os.Getenv("PAYLOAD_COMPRESSION"),
		os.Getenv("PAYLOAD_COMPRESSION_ATTRIBUTE"),
	)
	if err != nil {
		return nil, err
	}

	return &Config{
		PubSubProjectID:    projectID,
		PubSubTopicID:      topicID,
		WebhookSecrets:     secrets,
		ListenAddr:         listenAddr,
		ListenPort:         listenPort,
		LogLevel:           logLevel,
		PayloadCompression: payloadCompression,
	}, nil
}
