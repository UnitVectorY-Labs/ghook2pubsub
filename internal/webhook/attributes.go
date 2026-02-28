package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
)

// ExtractionResult contains the result of attribute extraction.
type ExtractionResult struct {
	Attributes  map[string]string
	Warnings    []string
	ParseFailed bool
}

type lookupStatus int

const (
	lookupMissing lookupStatus = iota
	lookupInvalid
	lookupFound
)

// ExtractAttributes extracts Pub/Sub message attributes from headers and payload.
func ExtractAttributes(headers http.Header, body []byte) ExtractionResult {
	attrs := make(map[string]string)
	var warnings []string

	setHeaderAttr(attrs, headers, "gh_delivery", "X-GitHub-Delivery")
	setHeaderAttr(attrs, headers, "gh_event", "X-GitHub-Event")
	setHeaderAttr(attrs, headers, "gh_hook_id", "X-GitHub-Hook-ID")
	setHeaderAttr(attrs, headers, "gh_target_type", "X-GitHub-Hook-Installation-Target-Type")
	setHeaderAttr(attrs, headers, "gh_target_id", "X-GitHub-Hook-Installation-Target-ID")

	payload, err := parsePayload(headers.Get("Content-Type"), body)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("payload_parse_failed: %v", err))
		return ExtractionResult{Attributes: attrs, Warnings: warnings, ParseFailed: true}
	}

	extractStringAttr(attrs, &warnings, payload, "action", "attribute_missing: action", []string{"action"})
	extractStringAttr(attrs, &warnings, payload, "organization", "attribute_missing: organization", []string{"organization", "login"})
	extractStringAttr(attrs, &warnings, payload, "repository", "attribute_missing: repository", []string{"repository", "full_name"})
	extractStringAttr(attrs, &warnings, payload, "sender", "attribute_missing: sender", []string{"sender", "login"})
	extractNumberAttr(attrs, &warnings, payload, "installation_id", "attribute_missing: installation_id", []string{"installation", "id"})
	extractStringAttr(attrs, &warnings, payload, "ref", "attribute_missing: ref", []string{"ref"})
	extractStringAttr(attrs, &warnings, payload, "ref_type", "attribute_missing: ref_type", []string{"ref_type"})

	extractOptionalStringAttr(attrs, payload, "base_ref", []string{"base_ref"}, []string{"pull_request", "base", "ref"})
	extractOptionalStringAttr(attrs, payload, "head_ref", []string{"head_ref"}, []string{"pull_request", "head", "ref"})
	extractOptionalStringAttr(attrs, payload, "state", []string{"state"}, []string{"deployment_status", "state"})
	extractOptionalStringAttr(attrs, payload, "status", []string{"status"}, []string{"check_run", "status"}, []string{"check_suite", "status"}, []string{"workflow_run", "status"}, []string{"workflow_job", "status"})
	extractOptionalStringAttr(attrs, payload, "conclusion", []string{"conclusion"}, []string{"check_run", "conclusion"}, []string{"check_suite", "conclusion"}, []string{"workflow_run", "conclusion"}, []string{"workflow_job", "conclusion"})

	return ExtractionResult{Attributes: attrs, Warnings: warnings, ParseFailed: false}
}

func setHeaderAttr(attrs map[string]string, headers http.Header, attrKey, headerKey string) {
	if v := headers.Get(headerKey); v != "" {
		attrs[attrKey] = v
	}
}

func parsePayload(contentType string, body []byte) (map[string]interface{}, error) {
	if payload, err := parseJSONPayload(body); err == nil {
		return payload, nil
	} else if !shouldAttemptFormPayload(contentType, body) {
		return nil, err
	}

	payload, err := parseFormPayload(body)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func parseJSONPayload(body []byte) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func parseFormPayload(body []byte) (map[string]interface{}, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	rawPayload := values.Get("payload")
	if rawPayload == "" {
		return nil, fmt.Errorf("form payload missing payload field")
	}

	return parseJSONPayload([]byte(rawPayload))
}

func shouldAttemptFormPayload(contentType string, body []byte) bool {
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil && mediaType == "application/x-www-form-urlencoded" {
		return true
	}

	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(trimmed, []byte("payload="))
}

func extractStringAttr(attrs map[string]string, warnings *[]string, payload map[string]interface{}, attrKey, warning string, paths ...[]string) {
	if value, ok, invalid := firstStringValue(payload, paths...); ok {
		attrs[attrKey] = value
	} else if invalid {
		*warnings = append(*warnings, warning)
	}
}

func extractOptionalStringAttr(attrs map[string]string, payload map[string]interface{}, attrKey string, paths ...[]string) {
	if value, ok, _ := firstStringValue(payload, paths...); ok {
		attrs[attrKey] = value
	}
}

func extractNumberAttr(attrs map[string]string, warnings *[]string, payload map[string]interface{}, attrKey, warning string, paths ...[]string) {
	if value, ok, invalid := firstNumberValue(payload, paths...); ok {
		attrs[attrKey] = value
	} else if invalid {
		*warnings = append(*warnings, warning)
	}
}

func firstStringValue(payload map[string]interface{}, paths ...[]string) (string, bool, bool) {
	invalid := false

	for _, path := range paths {
		value, status := lookupStringPath(payload, path...)
		switch status {
		case lookupFound:
			return value, true, false
		case lookupInvalid:
			invalid = true
		}
	}

	return "", false, invalid
}

func firstNumberValue(payload map[string]interface{}, paths ...[]string) (string, bool, bool) {
	invalid := false

	for _, path := range paths {
		value, status := lookupNumberPath(payload, path...)
		switch status {
		case lookupFound:
			return value, true, false
		case lookupInvalid:
			invalid = true
		}
	}

	return "", false, invalid
}

func lookupStringPath(payload map[string]interface{}, path ...string) (string, lookupStatus) {
	current := any(payload)

	for i, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", lookupInvalid
		}

		next, ok := m[key]
		if !ok {
			if i == 0 {
				return "", lookupMissing
			}
			return "", lookupInvalid
		}

		if next == nil {
			return "", lookupMissing
		}

		if i == len(path)-1 {
			s, ok := next.(string)
			if !ok || s == "" {
				return "", lookupInvalid
			}
			return s, lookupFound
		}

		current = next
	}

	return "", lookupMissing
}

func lookupNumberPath(payload map[string]interface{}, path ...string) (string, lookupStatus) {
	current := any(payload)

	for i, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", lookupInvalid
		}

		next, ok := m[key]
		if !ok {
			if i == 0 {
				return "", lookupMissing
			}
			return "", lookupInvalid
		}

		if next == nil {
			return "", lookupMissing
		}

		if i == len(path)-1 {
			switch value := next.(type) {
			case float64:
				return fmt.Sprintf("%d", int64(value)), lookupFound
			case json.Number:
				return value.String(), lookupFound
			default:
				return "", lookupInvalid
			}
		}

		current = next
	}

	return "", lookupMissing
}
