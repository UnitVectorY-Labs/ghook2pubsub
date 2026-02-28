package webhook

import (
	"net/http"
	"net/url"
	"testing"
)

func TestExtractAttributes_FullPayload(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-GitHub-Delivery", "delivery-123")
	headers.Set("X-GitHub-Event", "pull_request")
	headers.Set("X-GitHub-Hook-ID", "hook-456")
	headers.Set("X-GitHub-Hook-Installation-Target-Type", "organization")
	headers.Set("X-GitHub-Hook-Installation-Target-ID", "789")

	body := []byte(`{
		"action": "opened",
		"organization": {"login": "my-org"},
		"repository": {"full_name": "my-org/my-repo"},
		"sender": {"login": "octocat"},
		"installation": {"id": 12345},
		"ref": "refs/heads/main",
		"ref_type": "branch",
		"pull_request": {
			"base": {"ref": "main"},
			"head": {"ref": "feature-branch"}
		},
		"workflow_run": {
			"status": "completed",
			"conclusion": "failure"
		}
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}

	expected := map[string]string{
		"gh_delivery":     "delivery-123",
		"gh_event":        "pull_request",
		"gh_hook_id":      "hook-456",
		"gh_target_type":  "organization",
		"gh_target_id":    "789",
		"action":          "opened",
		"organization":    "my-org",
		"repository":      "my-org/my-repo",
		"sender":          "octocat",
		"installation_id": "12345",
		"ref":             "refs/heads/main",
		"ref_type":        "branch",
		"base_ref":        "main",
		"head_ref":        "feature-branch",
		"status":          "completed",
		"conclusion":      "failure",
	}

	for key, want := range expected {
		got, ok := result.Attributes[key]
		if !ok {
			t.Errorf("missing attribute %q", key)
			continue
		}
		if got != want {
			t.Errorf("attribute %q = %q, want %q", key, got, want)
		}
	}
}

func TestExtractAttributes_HeaderExtraction(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-GitHub-Delivery", "abc")
	headers.Set("X-GitHub-Event", "push")
	headers.Set("X-GitHub-Hook-ID", "789")
	headers.Set("X-GitHub-Hook-Installation-Target-Type", "repository")
	headers.Set("X-GitHub-Hook-Installation-Target-ID", "123")

	result := ExtractAttributes(headers, []byte(`{}`))
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}

	if result.Attributes["gh_delivery"] != "abc" {
		t.Errorf("gh_delivery = %q, want %q", result.Attributes["gh_delivery"], "abc")
	}
	if result.Attributes["gh_event"] != "push" {
		t.Errorf("gh_event = %q, want %q", result.Attributes["gh_event"], "push")
	}
	if result.Attributes["gh_hook_id"] != "789" {
		t.Errorf("gh_hook_id = %q, want %q", result.Attributes["gh_hook_id"], "789")
	}
	if result.Attributes["gh_target_type"] != "repository" {
		t.Errorf("gh_target_type = %q, want %q", result.Attributes["gh_target_type"], "repository")
	}
	if result.Attributes["gh_target_id"] != "123" {
		t.Errorf("gh_target_id = %q, want %q", result.Attributes["gh_target_id"], "123")
	}
}

func TestExtractAttributes_PartialPayloadWithoutWarnings(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	body := []byte(`{"action":"closed","sender":{"login":"user1"}}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["action"] != "closed" {
		t.Errorf("action = %q, want %q", result.Attributes["action"], "closed")
	}
	if result.Attributes["sender"] != "user1" {
		t.Errorf("sender = %q, want %q", result.Attributes["sender"], "user1")
	}
	if _, ok := result.Attributes["organization"]; ok {
		t.Errorf("expected organization to be absent, got %q", result.Attributes["organization"])
	}
	if _, ok := result.Attributes["repository"]; ok {
		t.Errorf("expected repository to be absent, got %q", result.Attributes["repository"])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for legitimately absent fields, got %v", result.Warnings)
	}
}

func TestExtractAttributes_PushPayloadWithoutActionWarning(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-GitHub-Event", "push")
	body := []byte(`{
		"repository": {"full_name": "UnitVectorY-Labs/arc-test"},
		"sender": {"login": "JaredHatfield"},
		"ref": "refs/heads/main",
		"base_ref": null
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["repository"] != "UnitVectorY-Labs/arc-test" {
		t.Errorf("repository = %q, want %q", result.Attributes["repository"], "UnitVectorY-Labs/arc-test")
	}
	if result.Attributes["sender"] != "JaredHatfield" {
		t.Errorf("sender = %q, want %q", result.Attributes["sender"], "JaredHatfield")
	}
	if result.Attributes["ref"] != "refs/heads/main" {
		t.Errorf("ref = %q, want %q", result.Attributes["ref"], "refs/heads/main")
	}
	if _, ok := result.Attributes["base_ref"]; ok {
		t.Errorf("expected base_ref to be absent, got %q", result.Attributes["base_ref"])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for push payload without action, got %v", result.Warnings)
	}
}

func TestExtractAttributes_PingPayloadWithoutRepositoryWarning(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-GitHub-Event", "ping")
	body := []byte(`{
		"organization": {"login": "UnitVectorY-Labs"},
		"sender": {"login": "JaredHatfield"}
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["organization"] != "UnitVectorY-Labs" {
		t.Errorf("organization = %q, want %q", result.Attributes["organization"], "UnitVectorY-Labs")
	}
	if result.Attributes["sender"] != "JaredHatfield" {
		t.Errorf("sender = %q, want %q", result.Attributes["sender"], "JaredHatfield")
	}
	if _, ok := result.Attributes["repository"]; ok {
		t.Errorf("expected repository to be absent, got %q", result.Attributes["repository"])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for ping payload without repository, got %v", result.Warnings)
	}
}

func TestExtractAttributes_WarnsForMalformedPresentFields(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	body := []byte(`{
		"action": 123,
		"organization": {},
		"repository": {},
		"sender": {},
		"installation": {},
		"ref": 123,
		"ref_type": 456
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}

	expectedWarnings := map[string]bool{
		"attribute_missing: action":          false,
		"attribute_missing: organization":    false,
		"attribute_missing: repository":      false,
		"attribute_missing: sender":          false,
		"attribute_missing: installation_id": false,
		"attribute_missing: ref":             false,
		"attribute_missing: ref_type":        false,
	}
	for _, warning := range result.Warnings {
		if _, ok := expectedWarnings[warning]; ok {
			expectedWarnings[warning] = true
		}
	}
	for warning, seen := range expectedWarnings {
		if !seen {
			t.Errorf("expected warning %q, got %v", warning, result.Warnings)
		}
	}
}

func TestExtractAttributes_StateStatusAndConclusionExtraction(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	body := []byte(`{
		"deployment_status": {"state": "failure"},
		"check_run": {
			"status": "completed",
			"conclusion": "timed_out"
		}
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["state"] != "failure" {
		t.Errorf("state = %q, want %q", result.Attributes["state"], "failure")
	}
	if result.Attributes["status"] != "completed" {
		t.Errorf("status = %q, want %q", result.Attributes["status"], "completed")
	}
	if result.Attributes["conclusion"] != "timed_out" {
		t.Errorf("conclusion = %q, want %q", result.Attributes["conclusion"], "timed_out")
	}
}

func TestExtractAttributes_FormEncodedPayload(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	body := []byte("payload=" + url.QueryEscape(`{"repository":{"full_name":"org/repo"},"sender":{"login":"octocat"}}`))

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["repository"] != "org/repo" {
		t.Errorf("repository = %q, want %q", result.Attributes["repository"], "org/repo")
	}
	if result.Attributes["sender"] != "octocat" {
		t.Errorf("sender = %q, want %q", result.Attributes["sender"], "octocat")
	}
}

func TestExtractAttributes_NonJSONPayload(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	result := ExtractAttributes(headers, []byte(`this is not json`))

	if !result.ParseFailed {
		t.Fatal("expected ParseFailed to be true for non-JSON body")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for parse failure")
	}
}

func TestExtractAttributes_EmptyBody(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	result := ExtractAttributes(headers, []byte(``))
	if !result.ParseFailed {
		t.Fatal("expected ParseFailed to be true for empty body")
	}
}
