package webhook

import (
	"net/http"
	"testing"
)

func TestExtractAttributes_FullPayload(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Delivery", "delivery-123")
	headers.Set("X-GitHub-Event", "pull_request")
	headers.Set("X-GitHub-Hook-ID", "hook-456")

	body := []byte(`{
		"action": "opened",
		"organization": {"login": "my-org"},
		"repository": {"full_name": "my-org/my-repo"},
		"sender": {"login": "octocat"},
		"installation": {"id": 12345},
		"issue": {"number": 42},
		"pull_request": {
			"number": 99,
			"base": {"ref": "main"},
			"head": {"ref": "feature-branch"}
		},
		"ref": "refs/heads/main"
	}`)

	result := ExtractAttributes(headers, body)

	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}

	expected := map[string]string{
		"delivery":        "delivery-123",
		"gh_event":        "pull_request",
		"gh_hook_id":      "hook-456",
		"action":          "opened",
		"org":             "my-org",
		"repo":            "my-org/my-repo",
		"sender":          "octocat",
		"installation_id": "12345",
		"issue_number":    "42",
		"pr_number":       "99",
		"ref":             "refs/heads/main",
		"base_ref":        "main",
		"head_ref":        "feature-branch",
	}

	for k, want := range expected {
		got, ok := result.Attributes[k]
		if !ok {
			t.Errorf("missing attribute %q", k)
			continue
		}
		if got != want {
			t.Errorf("attribute %q = %q, want %q", k, got, want)
		}
	}
}

func TestExtractAttributes_HeaderExtraction(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Delivery", "abc")
	headers.Set("X-GitHub-Event", "push")
	headers.Set("X-GitHub-Hook-ID", "789")

	body := []byte(`{}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}

	if result.Attributes["delivery"] != "abc" {
		t.Errorf("delivery = %q, want %q", result.Attributes["delivery"], "abc")
	}
	if result.Attributes["gh_event"] != "push" {
		t.Errorf("gh_event = %q, want %q", result.Attributes["gh_event"], "push")
	}
	if result.Attributes["gh_hook_id"] != "789" {
		t.Errorf("gh_hook_id = %q, want %q", result.Attributes["gh_hook_id"], "789")
	}
}

func TestExtractAttributes_PartialPayload(t *testing.T) {
	headers := http.Header{}
	body := []byte(`{"action": "closed", "sender": {"login": "user1"}}`)

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
	// org and repo should be missing, generating warnings
	if _, ok := result.Attributes["org"]; ok {
		t.Error("expected org to be missing")
	}
	if _, ok := result.Attributes["repo"]; ok {
		t.Error("expected repo to be missing")
	}
	if len(result.Warnings) < 2 {
		t.Errorf("expected at least 2 warnings for missing org and repo, got %d", len(result.Warnings))
	}
}

func TestExtractAttributes_NonJSONPayload(t *testing.T) {
	headers := http.Header{}
	body := []byte(`this is not json`)

	result := ExtractAttributes(headers, body)
	if !result.ParseFailed {
		t.Fatal("expected ParseFailed to be true for non-JSON body")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for parse failure")
	}
}

func TestExtractAttributes_EmptyBody(t *testing.T) {
	headers := http.Header{}
	body := []byte(``)

	result := ExtractAttributes(headers, body)
	if !result.ParseFailed {
		t.Fatal("expected ParseFailed to be true for empty body")
	}
}

func TestExtractAttributes_ActionNoOrgRepo(t *testing.T) {
	headers := http.Header{}
	body := []byte(`{"action": "created"}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}
	if result.Attributes["action"] != "created" {
		t.Errorf("action = %q, want %q", result.Attributes["action"], "created")
	}

	hasOrgWarning := false
	hasRepoWarning := false
	for _, w := range result.Warnings {
		if w == "attribute_missing: organization" {
			hasOrgWarning = true
		}
		if w == "attribute_missing: repository" {
			hasRepoWarning = true
		}
	}
	if !hasOrgWarning {
		t.Error("expected warning about missing organization")
	}
	if !hasRepoWarning {
		t.Error("expected warning about missing repository")
	}
}

func TestExtractAttributes_NumberConversion(t *testing.T) {
	headers := http.Header{}
	body := []byte(`{
		"installation": {"id": 98765},
		"issue": {"number": 100},
		"pull_request": {"number": 200}
	}`)

	result := ExtractAttributes(headers, body)
	if result.ParseFailed {
		t.Fatal("expected parse to succeed")
	}

	if result.Attributes["installation_id"] != "98765" {
		t.Errorf("installation_id = %q, want %q", result.Attributes["installation_id"], "98765")
	}
	if result.Attributes["issue_number"] != "100" {
		t.Errorf("issue_number = %q, want %q", result.Attributes["issue_number"], "100")
	}
	if result.Attributes["pr_number"] != "200" {
		t.Errorf("pr_number = %q, want %q", result.Attributes["pr_number"], "200")
	}
}

func TestExtractAttributes_ParseFailedFlag(t *testing.T) {
	headers := http.Header{}
	body := []byte(`{invalid json}`)

	result := ExtractAttributes(headers, body)
	if !result.ParseFailed {
		t.Fatal("expected ParseFailed to be true for invalid JSON")
	}
	if result.Attributes == nil {
		t.Fatal("expected Attributes map to be non-nil even on parse failure")
	}
}
