package publisher

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/klauspost/compress/zstd"
)

// CompressionAlgorithm identifies the payload compression applied before publish.
type CompressionAlgorithm string

const (
	CompressionNone CompressionAlgorithm = "none"
	CompressionGzip CompressionAlgorithm = "gzip"
	CompressionZstd CompressionAlgorithm = "zstd"
)

// Publisher defines the interface for publishing messages.
type Publisher interface {
	Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error)
	Close() error
}

// CompressionConfig describes optional payload compression and attribute emission.
type CompressionConfig struct {
	Algorithm     CompressionAlgorithm
	Level         int
	AttributeName string
}

// Enabled reports whether payload compression is configured.
func (c CompressionConfig) Enabled() bool {
	return c.Algorithm != CompressionNone
}

// String returns the normalized configuration string for logs and diagnostics.
func (c CompressionConfig) String() string {
	if !c.Enabled() {
		return string(CompressionNone)
	}

	return fmt.Sprintf("%s:%d", c.Algorithm, c.Level)
}

// ParseCompressionConfig validates and normalizes the configured compression specification.
func ParseCompressionConfig(spec string, attributeName string) (CompressionConfig, error) {
	normalizedAttributeName := strings.TrimSpace(attributeName)
	normalizedSpec := strings.ToLower(strings.TrimSpace(spec))
	if normalizedSpec == "" || normalizedSpec == string(CompressionNone) {
		return CompressionConfig{
			Algorithm:     CompressionNone,
			AttributeName: normalizedAttributeName,
		}, nil
	}

	parts := strings.SplitN(normalizedSpec, ":", 2)
	if len(parts) != 2 {
		return CompressionConfig{}, fmt.Errorf("unsupported payload compression %q: expected format <algorithm>:<level>, such as gzip:6 or zstd:3", spec)
	}

	level, err := strconv.Atoi(parts[1])
	if err != nil {
		return CompressionConfig{}, fmt.Errorf("unsupported payload compression %q: invalid compression level %q", spec, parts[1])
	}

	switch CompressionAlgorithm(parts[0]) {
	case CompressionGzip:
		if level < gzip.BestSpeed || level > gzip.BestCompression {
			return CompressionConfig{}, fmt.Errorf("unsupported payload compression %q: gzip level must be between %d and %d", spec, gzip.BestSpeed, gzip.BestCompression)
		}
		return CompressionConfig{
			Algorithm:     CompressionGzip,
			Level:         level,
			AttributeName: normalizedAttributeName,
		}, nil
	case CompressionZstd:
		if level < 1 {
			return CompressionConfig{}, fmt.Errorf("unsupported payload compression %q: zstd level must be a positive integer", spec)
		}
		return CompressionConfig{
			Algorithm:     CompressionZstd,
			Level:         level,
			AttributeName: normalizedAttributeName,
		}, nil
	default:
		return CompressionConfig{}, fmt.Errorf("unsupported payload compression %q: supported values are none, gzip:<level>, zstd:<level>", spec)
	}
}

// PubSubPublisher implements Publisher using Google Cloud Pub/Sub.
type PubSubPublisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// NewPubSubPublisher creates a new PubSubPublisher for the given project and topic.
func NewPubSubPublisher(ctx context.Context, projectID, topicID string) (*PubSubPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicID)

	return &PubSubPublisher{
		client: client,
		topic:  topic,
	}, nil
}

// Publish sends data with attributes to Pub/Sub and returns the server-assigned message ID.
func (p *PubSubPublisher) Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error) {
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	})

	serverID, err := result.Get(ctx)
	if err != nil {
		return "", err
	}

	return serverID, nil
}

// Close stops the topic and closes the underlying client.
func (p *PubSubPublisher) Close() error {
	p.topic.Stop()
	return p.client.Close()
}

// CompressingPublisher wraps another publisher and compresses payloads before publishing.
type CompressingPublisher struct {
	next   Publisher
	config CompressionConfig
}

// NewCompressingPublisher creates a publisher wrapper that optionally compresses payloads.
func NewCompressingPublisher(next Publisher, config CompressionConfig) Publisher {
	if !config.Enabled() {
		return next
	}

	return &CompressingPublisher{
		next:   next,
		config: config,
	}
}

// Publish compresses the payload and marks the message attributes when compression is enabled.
func (p *CompressingPublisher) Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error) {
	compressed, err := compressPayload(data, p.config)
	if err != nil {
		return "", err
	}

	return p.next.Publish(ctx, compressed, cloneAttributesWithCompression(attributes, p.config))
}

// Close closes the wrapped publisher.
func (p *CompressingPublisher) Close() error {
	return p.next.Close()
}

func compressPayload(data []byte, config CompressionConfig) ([]byte, error) {
	switch config.Algorithm {
	case CompressionGzip:
		var buf bytes.Buffer
		writer, err := gzip.NewWriterLevel(&buf, config.Level)
		if err != nil {
			return nil, fmt.Errorf("create gzip writer: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			writer.Close()
			return nil, fmt.Errorf("gzip payload: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("finalize gzip payload: %w", err)
		}
		return buf.Bytes(), nil
	case CompressionZstd:
		var buf bytes.Buffer
		writer, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(config.Level)))
		if err != nil {
			return nil, fmt.Errorf("create zstd writer: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			writer.Close()
			return nil, fmt.Errorf("zstd payload: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("finalize zstd payload: %w", err)
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm %q", config.Algorithm)
	}
}

func cloneAttributesWithCompression(attributes map[string]string, config CompressionConfig) map[string]string {
	if config.AttributeName == "" {
		return attributes
	}

	cloned := make(map[string]string, len(attributes)+1)
	for key, value := range attributes {
		cloned[key] = value
	}
	cloned[config.AttributeName] = string(config.Algorithm)
	return cloned
}
