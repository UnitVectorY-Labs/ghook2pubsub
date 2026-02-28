# Pub/Sub Message Attributes

Every webhook payload published to Pub/Sub carries a set of attributes extracted from the incoming GitHub delivery headers and JSON payload. These attributes are intended for routing and filtering, so the contract focuses on low-cardinality fields that are stable across event families.

## Stability Guarantee

- Attribute keys listed below are stable and will not be renamed or removed without a major version bump.
- Adding new attribute keys is backward-compatible.
- Removing an existing attribute key is a breaking change.

## Header-Derived Attributes

These attributes are copied directly from GitHub delivery headers.

| Attribute Key | Source Header | Applies To | Description |
|---------------|---------------|------------|-------------|
| `gh_delivery` | `X-GitHub-Delivery` | All deliveries | Unique delivery GUID assigned by GitHub. Useful for tracing and deduplication. |
| `gh_event` | `X-GitHub-Event` | All deliveries | GitHub event type such as `push`, `pull_request`, or `workflow_run`. |
| `gh_hook_id` | `X-GitHub-Hook-ID` | Deliveries that include the header | Numeric identifier of the webhook configuration that sent the delivery. |
| `gh_target_type` | `X-GitHub-Hook-Installation-Target-Type` | Deliveries that include the header | Target type for the webhook installation scope, such as `repository` or `organization`. |
| `gh_target_id` | `X-GitHub-Hook-Installation-Target-ID` | Deliveries that include the header | Stable identifier for the webhook installation target. |

## Payload-Derived Attributes

These attributes are extracted from the webhook payload on a best-effort basis.

| Attribute Key | Source | Applies To | Description |
|---------------|--------|------------|-------------|
| `action` | `.action` | Events with action semantics | Event subtype such as `opened`, `closed`, `completed`, or `created`. |
| `organization` | `.organization.login` | Deliveries that include a top-level `organization` object | Organization login associated with the delivery. |
| `repository` | `.repository.full_name` | Deliveries that include a top-level `repository` object | Repository full name in `owner/repo` form. |
| `sender` | `.sender.login` | Deliveries that include a top-level `sender` object | Login of the GitHub user that triggered or is associated with the delivery. |
| `installation_id` | `.installation.id` | GitHub App deliveries that include an installation object | GitHub App installation identifier formatted as a string. |
| `ref` | `.ref` | Ref-oriented events such as `push`, `create`, and `delete` | Fully qualified Git ref such as `refs/heads/main` or `refs/tags/v1.2.3`. |
| `ref_type` | `.ref_type` | Ref creation and deletion events | Ref category, typically `branch` or `tag`. |
| `base_ref` | `.base_ref`, fallback to `.pull_request.base.ref` | Events that carry a base branch reference | Base branch reference used for branch-targeted routing. |
| `head_ref` | `.head_ref`, fallback to `.pull_request.head.ref` | Events that carry a head branch reference | Head branch reference used for source-branch routing. |
| `state` | `.state`, fallback to `.deployment_status.state` | Events that carry a coarse state value | Broad state value such as `pending`, `success`, `failure`, or `error`. |
| `status` | `.status`, fallback to event objects such as `.check_run.status`, `.check_suite.status`, `.workflow_run.status`, `.workflow_job.status` | Events that carry lifecycle status | Lifecycle status such as `queued`, `in_progress`, or `completed`. |
| `conclusion` | `.conclusion`, fallback to event objects such as `.check_run.conclusion`, `.check_suite.conclusion`, `.workflow_run.conclusion`, `.workflow_job.conclusion` | Events that carry a terminal result | Terminal result such as `success`, `failure`, `cancelled`, or `timed_out`. |

## Best-Effort Extraction

- GitHub deliveries sent as `application/json` and `application/x-www-form-urlencoded` are both supported.
- If a payload field is absent because that event type does not include it, the corresponding attribute is omitted.
- If GitHub includes a relevant key or object but it cannot be converted into the documented attribute value, the payload is still published and a structured warning is emitted.
- If the payload cannot be parsed, the message is still published with any header-derived attributes that were available.

## Value Constraints

- All attribute values are UTF-8 strings.
- Numeric payload values are converted to decimal string form.
- Attributes are intended to remain small, stable, and suitable for Pub/Sub filtering.
