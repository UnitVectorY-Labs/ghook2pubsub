package webhook

import "sync/atomic"

// Metrics holds thread-safe atomic counters for webhook processing.
type Metrics struct {
	TotalRequests               atomic.Int64
	SignatureFailures           atomic.Int64
	PublishSuccesses            atomic.Int64
	PublishFailures             atomic.Int64
	PayloadParseFailures        atomic.Int64
	AttributeExtractionWarnings atomic.Int64
}
