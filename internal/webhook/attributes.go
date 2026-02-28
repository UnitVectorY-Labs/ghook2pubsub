package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ExtractionResult contains the result of attribute extraction.
type ExtractionResult struct {
	Attributes  map[string]string
	Warnings    []string
	ParseFailed bool
}

// ExtractAttributes extracts Pub/Sub message attributes from headers and payload.
func ExtractAttributes(headers http.Header, body []byte) ExtractionResult {
	attrs := make(map[string]string)
	var warnings []string

	// Header-derived attributes
	if v := headers.Get("X-GitHub-Delivery"); v != "" {
		attrs["delivery"] = v
	}
	if v := headers.Get("X-GitHub-Event"); v != "" {
		attrs["gh_event"] = v
	}
	if v := headers.Get("X-GitHub-Hook-ID"); v != "" {
		attrs["gh_hook_id"] = v
	}

	// Payload-derived attributes
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		warnings = append(warnings, fmt.Sprintf("payload_parse_failed: %v", err))
		return ExtractionResult{Attributes: attrs, Warnings: warnings, ParseFailed: true}
	}

	setString(attrs, payload, "action", "action")
	setNested(attrs, payload, "org", "organization", "login")
	setNested(attrs, payload, "repo", "repository", "full_name")
	setNested(attrs, payload, "sender", "sender", "login")
	setNestedNumber(attrs, payload, "installation_id", "installation", "id")
	setNestedNumber(attrs, payload, "issue_number", "issue", "number")
	setNestedNumber(attrs, payload, "pr_number", "pull_request", "number")
	setString(attrs, payload, "ref", "ref")
	setDeepNested(attrs, payload, "base_ref", "pull_request", "base", "ref")
	setDeepNested(attrs, payload, "head_ref", "pull_request", "head", "ref")

	// Warn about missing common fields
	if _, ok := attrs["org"]; !ok {
		warnings = append(warnings, "attribute_missing: organization")
	}
	if _, ok := attrs["repo"]; !ok {
		warnings = append(warnings, "attribute_missing: repository")
	}
	if _, ok := attrs["action"]; !ok {
		warnings = append(warnings, "attribute_missing: action")
	}

	return ExtractionResult{Attributes: attrs, Warnings: warnings, ParseFailed: false}
}

func setString(attrs map[string]string, payload map[string]interface{}, attrKey, payloadKey string) {
	if v, ok := payload[payloadKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			attrs[attrKey] = s
		}
	}
}

func setNested(attrs map[string]string, payload map[string]interface{}, attrKey, outerKey, innerKey string) {
	if outer, ok := payload[outerKey]; ok {
		if m, ok := outer.(map[string]interface{}); ok {
			if v, ok := m[innerKey]; ok {
				if s, ok := v.(string); ok && s != "" {
					attrs[attrKey] = s
				}
			}
		}
	}
}

func setNestedNumber(attrs map[string]string, payload map[string]interface{}, attrKey, outerKey, innerKey string) {
	if outer, ok := payload[outerKey]; ok {
		if m, ok := outer.(map[string]interface{}); ok {
			if v, ok := m[innerKey]; ok {
				if n, ok := v.(float64); ok {
					attrs[attrKey] = fmt.Sprintf("%d", int64(n))
				}
			}
		}
	}
}

func setDeepNested(attrs map[string]string, payload map[string]interface{}, attrKey, key1, key2, key3 string) {
	if outer, ok := payload[key1]; ok {
		if m1, ok := outer.(map[string]interface{}); ok {
			if mid, ok := m1[key2]; ok {
				if m2, ok := mid.(map[string]interface{}); ok {
					if v, ok := m2[key3]; ok {
						if s, ok := v.(string); ok && s != "" {
							attrs[attrKey] = s
						}
					}
				}
			}
		}
	}
}
