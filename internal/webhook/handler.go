package webhook

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/publisher"
)

// Handler processes incoming GitHub webhook requests.
type Handler struct {
	pub     publisher.Publisher
	secrets []string
	metrics *Metrics
}

// NewHandler creates a new webhook Handler.
func NewHandler(pub publisher.Publisher, secrets []string, metrics *Metrics) *Handler {
	return &Handler{
		pub:     pub,
		secrets: secrets,
		metrics: metrics,
	}
}

// ServeHTTP handles incoming webhook requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.metrics.TotalRequests.Add(1)

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Build log context from headers
	delivery := r.Header.Get("X-GitHub-Delivery")
	ghEvent := r.Header.Get("X-GitHub-Event")
	ghHookID := r.Header.Get("X-GitHub-Hook-ID")
	logAttrs := []any{}
	if delivery != "" {
		logAttrs = append(logAttrs, "delivery", delivery)
	}
	if ghEvent != "" {
		logAttrs = append(logAttrs, "gh_event", ghEvent)
	}
	if ghHookID != "" {
		logAttrs = append(logAttrs, "gh_hook_id", ghHookID)
	}

	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" {
		h.metrics.SignatureFailures.Add(1)
		slog.Warn("signature verification failed", append(logAttrs, "reason", "signature_missing")...)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !VerifySignature(body, signatureHeader, h.secrets) {
		h.metrics.SignatureFailures.Add(1)
		slog.Warn("signature verification failed", append(logAttrs, "reason", "signature_invalid")...)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	attrs, warnings := ExtractAttributes(r.Header, body)
	if len(warnings) > 0 {
		h.metrics.AttributeExtractionWarnings.Add(int64(len(warnings)))
		for _, warn := range warnings {
			slog.Debug("attribute extraction warning", append(logAttrs, "warning", warn)...)
		}
	}

	serverID, err := h.pub.Publish(r.Context(), body, attrs)
	if err != nil {
		h.metrics.PublishFailures.Add(1)
		slog.Error("publish failed", append(logAttrs, "reason", "pubsub_publish_failed", "error", err.Error())...)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.metrics.PublishSuccesses.Add(1)
	slog.Info("webhook published", append(logAttrs, "server_message_id", serverID)...)
	w.WriteHeader(http.StatusNoContent)
}

// HealthHandler returns a simple health check response.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
