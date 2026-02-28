# Pub/Sub Message Attributes

Every webhook payload published to Pub/Sub carries a set of **attributes** (key-value string metadata) extracted from the incoming HTTP headers and JSON body. Subscribers can use these attributes for filtering and routing without parsing the message body.

## Stability Guarantee

- Attribute **keys** listed below are considered stable and will not be renamed or removed without a major version bump.
- **Adding** new attribute keys is backward-compatible.
- **Removing** an existing key is a breaking change.

## Header-Derived Attributes

These attributes are copied directly from the GitHub webhook HTTP headers.

| Attribute Key | Source Header | Description |
|---------------|---------------|-------------|
| `delivery` | `X-GitHub-Delivery` | Unique delivery GUID assigned by GitHub. |
| `gh_event` | `X-GitHub-Event` | Event type (e.g. `push`, `pull_request`). |
| `gh_hook_id` | `X-GitHub-Hook-ID` | Numeric ID of the webhook configuration. |

## Payload-Derived Attributes

These attributes are extracted from the JSON body on a best-effort basis.

| Attribute Key | JSON Path | Description |
|---------------|-----------|-------------|
| `action` | `.action` | Action that triggered the event (e.g. `opened`, `closed`). |
| `org` | `.organization.login` | Organization login name. |
| `repo` | `.repository.full_name` | Full repository name (`owner/repo`). |
| `sender` | `.sender.login` | GitHub username of the user who triggered the event. |
| `installation_id` | `.installation.id` | GitHub App installation ID (numeric, formatted as string). |
| `issue_number` | `.issue.number` | Issue number (numeric, formatted as string). |
| `pr_number` | `.pull_request.number` | Pull request number (numeric, formatted as string). |
| `ref` | `.ref` | Git ref (branch or tag) for push/create/delete events. |
| `base_ref` | `.pull_request.base.ref` | Base branch of a pull request. |
| `head_ref` | `.pull_request.head.ref` | Head branch of a pull request. |

## Value Constraints

- All attribute values are valid UTF-8 strings.
- Numeric JSON values (IDs, numbers) are formatted as decimal integer strings.
- Values are small and bounded by the source data (GitHub header values and short JSON fields).

## Best-Effort Extraction

Attribute extraction is **best-effort**:

- If the JSON body cannot be parsed, only header-derived attributes are included. The payload is still published.
- If a JSON field is absent or has an unexpected type, the corresponding attribute is simply omitted.
- Missing common fields (`organization`, `repository`, `action`) produce warnings in the structured logs but do not block publishing.

## Example Attribute Maps

### `push` event

```json
{
  "delivery": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "gh_event": "push",
  "gh_hook_id": "123456789",
  "repo": "octocat/Hello-World",
  "org": "octocat-org",
  "sender": "octocat",
  "ref": "refs/heads/main"
}
```

### `pull_request` event (opened)

```json
{
  "delivery": "f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "gh_event": "pull_request",
  "gh_hook_id": "123456789",
  "action": "opened",
  "repo": "octocat/Hello-World",
  "org": "octocat-org",
  "sender": "octocat",
  "pr_number": "42",
  "base_ref": "main",
  "head_ref": "feature-branch"
}
```

### `issues` event (closed)

```json
{
  "delivery": "11223344-5566-7788-99aa-bbccddeeff00",
  "gh_event": "issues",
  "gh_hook_id": "123456789",
  "action": "closed",
  "repo": "octocat/Hello-World",
  "org": "octocat-org",
  "sender": "octocat",
  "issue_number": "7"
}
```
