package publisher

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
)

type recordingPublisher struct {
	data       []byte
	attributes map[string]string
}

func (r *recordingPublisher) Publish(_ context.Context, data []byte, attributes map[string]string) (string, error) {
	r.data = data
	r.attributes = attributes
	return "message-id", nil
}

func (r *recordingPublisher) Close() error {
	return nil
}

func TestParseCompressionConfig(t *testing.T) {
	tests := []struct {
		name          string
		spec          string
		attributeName string
		want          CompressionConfig
		wantErr       bool
	}{
		{
			name: "default empty",
			want: CompressionConfig{Algorithm: CompressionNone},
		},
		{
			name: "explicit none",
			spec: "none",
			want: CompressionConfig{Algorithm: CompressionNone},
		},
		{
			name: "gzip with level",
			spec: "gzip:6",
			want: CompressionConfig{Algorithm: CompressionGzip, Level: 6},
		},
		{
			name:          "zstd with attribute",
			spec:          "ZSTD:3",
			attributeName: " compression ",
			want:          CompressionConfig{Algorithm: CompressionZstd, Level: 3, AttributeName: "compression"},
		},
		{
			name:    "gzip missing level",
			spec:    "gzip",
			wantErr: true,
		},
		{
			name:    "gzip invalid level",
			spec:    "gzip:0",
			wantErr: true,
		},
		{
			name:    "zstd invalid level",
			spec:    "zstd:0",
			wantErr: true,
		},
		{
			name:    "unsupported algorithm",
			spec:    "brotli:5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCompressionConfig(tt.spec, tt.attributeName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("config = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewCompressingPublisherReturnsOriginalForNone(t *testing.T) {
	base := &recordingPublisher{}

	got := NewCompressingPublisher(base, CompressionConfig{Algorithm: CompressionNone})

	if got != base {
		t.Fatal("expected base publisher to be returned unchanged")
	}
}

func TestCompressingPublisherGzipWithoutAttribute(t *testing.T) {
	base := &recordingPublisher{}
	pub := NewCompressingPublisher(base, CompressionConfig{
		Algorithm: CompressionGzip,
		Level:     6,
	})
	original := []byte(`{"repository":"UnitVectorY-Labs/ghook2pubsub","action":"opened","sender":"octocat"}`)
	attributes := map[string]string{"gh_event": "issues"}

	if _, err := pub.Publish(context.Background(), original, attributes); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if bytes.Equal(base.data, original) {
		t.Fatal("expected compressed payload to differ from original")
	}

	reader, err := gzip.NewReader(bytes.NewReader(base.data))
	if err != nil {
		t.Fatalf("create gzip reader: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip payload: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Fatalf("decompressed payload = %q, want %q", decompressed, original)
	}

	if got := base.attributes["gh_event"]; got != "issues" {
		t.Fatalf("gh_event = %q, want %q", got, "issues")
	}
	if got := base.attributes["compression"]; got != "" {
		t.Fatalf("compression = %q, want empty", got)
	}
}

func TestCompressingPublisherZstdWithAttribute(t *testing.T) {
	base := &recordingPublisher{}
	pub := NewCompressingPublisher(base, CompressionConfig{
		Algorithm:     CompressionZstd,
		Level:         3,
		AttributeName: "compression",
	})
	original := []byte(`{"repository":"UnitVectorY-Labs/ghook2pubsub","action":"opened","sender":"octocat"}`)
	attributes := map[string]string{"gh_event": "issues"}

	if _, err := pub.Publish(context.Background(), original, attributes); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	reader, err := zstd.NewReader(bytes.NewReader(base.data))
	if err != nil {
		t.Fatalf("create zstd reader: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read zstd payload: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Fatalf("decompressed payload = %q, want %q", decompressed, original)
	}

	if got := base.attributes["compression"]; got != string(CompressionZstd) {
		t.Fatalf("compression = %q, want %q", got, CompressionZstd)
	}
	if got := attributes["compression"]; got != "" {
		t.Fatalf("expected original attributes map to remain unchanged, got %q", got)
	}
}
