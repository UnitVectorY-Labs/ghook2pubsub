# Usage

## Connect GitHub

1. Start the service using [Installation](INSTALL.md).
2. In the repository or organization webhook settings, add the public URL `https://your-host/webhook`.
3. Select `application/json` as the content type and enter the same secret configured in `WEBHOOK_SECRETS`.
4. Choose the events your subscribers need and enable the webhook.
5. Inspect the initial `ping` delivery in GitHub's recent deliveries. A successful publish returns `204 No Content`.

The service forwards every valid delivery, including `ping`; filter events using Pub/Sub subscriptions or downstream consumers. Form-encoded deliveries are supported for attribute extraction, but the published body remains form encoded. Prefer JSON for downstream JSON consumers.

## Verify delivery

Look for the JSON log entry `webhook published` on stdout. Match `gh_delivery` to GitHub's delivery ID; `server_message_id` identifies the published Pub/Sub message. Inspect a test subscription as shown in [Examples](EXAMPLES.md).

A `204` means publishing succeeded, not that any subscriber completed processing. Subscribers should tolerate duplicates and can use `gh_delivery` to identify repeated deliveries. The service has no durable local buffer. For failed deliveries, inspect GitHub's delivery history and explicitly redeliver after correcting the issue; do not rely on automatic redelivery.

## Troubleshooting

| Result | What to check |
| --- | --- |
| Startup configuration error | Required variables and compression syntax in [Configuration](CONFIGURATION.md). |
| `401 Unauthorized` | `User-Agent` must start with `GitHub-Hookshot/` and `X-Hub-Signature-256` must be present. |
| `403 Forbidden` | GitHub's secret must match one configured secret. A proxy must preserve the signed body bytes. |
| `405 Method Not Allowed` | Send `POST` to `/webhook`. |
| `500 Internal Server Error` | Inspect logs for body-read or publish failures; check credentials, IAM, topic ID, and connectivity. |
| `204` but no consumer activity | Check subscription topic, filter attributes, consumer logs, and compression compatibility. |
| Attribute extraction warning | Missing or malformed JSON fields may omit attributes; a valid signed body is still published. |

Logs use JSON on stdout. Set `LOG_LEVEL=debug` for more verbose logging. Correlate `gh_delivery`, `gh_event`, and `gh_hook_id` when present. The service stops accepting requests on `SIGTERM` or `SIGINT` and allows up to 15 seconds for active HTTP requests to finish.
