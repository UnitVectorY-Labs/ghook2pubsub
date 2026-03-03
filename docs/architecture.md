# Architecture

## Purpose and Scope

ghook2pubsub is a **single-purpose ingestion service**. It accepts GitHub webhook HTTP requests, verifies their authenticity, and publishes the raw payload to a Google Cloud Pub/Sub topic with extracted metadata attributes. Downstream consumers subscribe to the topic for further processing.

## Non-Goals

- **No filtering** – every valid webhook is published; subscribers decide what to act on.
- **No queuing** – the service is stateless and does not buffer messages internally.
- **No semantic payload mutation** – the webhook content is not rewritten. The only optional payload transformation is transport compression before Pub/Sub publish.

## Request Processing Flow

1. GitHub sends an HTTP `POST` to `/webhook`.
2. The `User-Agent` header is checked to ensure it starts with `GitHub-Hookshot/`. Requests that fail this check are rejected with `401`.
3. The server reads the full request body.
4. The `X-Hub-Signature-256` header is validated against the configured secrets using HMAC-SHA256. If the header is missing the request is rejected with `401`; if no secret matches, it is rejected with `403`.
5. Attributes are extracted from the HTTP headers and JSON body (see [attributes.md](attributes.md)).
6. The body and attributes are published to the configured Pub/Sub topic. If `PAYLOAD_COMPRESSION` is enabled, the body is compressed immediately before publish. If `PAYLOAD_COMPRESSION_ATTRIBUTE` is also set, that attribute is added with the compression algorithm name.
7. On successful publish the server responds with `204 No Content`.

## HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/webhook` | Webhook ingestion endpoint. |
| `GET` | `/healthz` | Health check. Returns `200` with `{"status":"ok"}`. |

## HTTP Response Codes

| Code | Meaning |
|------|---------|
| `204 No Content` | Webhook received, verified, and published successfully. |
| `401 Unauthorized` | `User-Agent` is not a GitHub Hookshot value or `X-Hub-Signature-256` is missing. |
| `403 Forbidden` | Signature verification failed (no configured secret matched). |
| `405 Method Not Allowed` | Request used an HTTP method other than `POST` on `/webhook`. |
| `500 Internal Server Error` | Body read failure or Pub/Sub publish failure. |

## Error Handling

- **Early request rejections** are logged at `warn` level with the reason (`user_agent_invalid`, `signature_missing`, or `signature_invalid`) and the request is rejected. The payload is never published.
- **Pub/Sub publish errors** are logged at `error` level and the server returns `500`, signaling GitHub to retry the delivery.
- **Payload parse failures** (malformed JSON) and malformed attribute structures are logged as warnings. The payload is still published with only the attributes that could be extracted; this ensures no data is silently dropped.

## Delivery Semantics

From GitHub's perspective the service provides **at-least-once** delivery semantics. GitHub retries webhook deliveries that do not receive a `2xx` response, and the service only returns `204` after the Pub/Sub publish call succeeds. If the network drops the `204` response after a successful publish, GitHub may retry and the message could be published again. Subscribers should be prepared to handle duplicate deliveries (the `gh_delivery` attribute can be used for deduplication).

## Structured Logging

All log output is JSON (via Go's `log/slog` with `JSONHandler`), written to stdout. Key fields included in log entries:

| Field | Description |
|-------|-------------|
| `gh_delivery` | GitHub delivery GUID (when present). |
| `gh_event` | GitHub event type (when present). |
| `gh_hook_id` | GitHub hook ID (when present). |
| `server_message_id` | Pub/Sub server-assigned message ID (on successful publish). |
| `reason` | Machine-readable failure reason (on errors/warnings). |
| `error` | Error message (on failures). |
| `warning` | Attribute extraction warning detail. |

The log level is configurable via the `LOG_LEVEL` environment variable (see [configuration.md](configuration.md)).

## Graceful Shutdown

The server listens for `SIGTERM` and `SIGINT` signals. On receipt it stops accepting new connections and allows up to 15 seconds for in-flight requests to complete before exiting.
