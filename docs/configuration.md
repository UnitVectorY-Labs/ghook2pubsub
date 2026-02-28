# Configuration

ghook2pubsub is configured entirely through environment variables.

## Environment Variables

| Name | Required | Default | Description |
|------|----------|---------|-------------|
| `PUBSUB_PROJECT_ID` | Yes | — | GCP project ID that owns the Pub/Sub topic. |
| `PUBSUB_TOPIC_ID` | Yes | — | Pub/Sub topic ID to publish webhook payloads to. |
| `WEBHOOK_SECRETS` | Yes | — | Comma-separated list of GitHub webhook secrets used for HMAC-SHA256 signature verification. |
| `LISTEN_ADDR` | No | `0.0.0.0` | Address the HTTP server binds to. |
| `LISTEN_PORT` | No | `8080` | Port the HTTP server listens on. |
| `LOG_LEVEL` | No | `info` | Log verbosity. Accepted values: `debug`, `info`, `warn`, `error`. |

The application exits immediately on startup if any required variable is missing or empty.

## Secret Rotation

`WEBHOOK_SECRETS` accepts multiple secrets separated by commas. During signature verification the application tries each secret in order and accepts the request if **any** secret produces a valid HMAC-SHA256 match.

This enables zero-downtime secret rotation:

1. Generate a new secret.
2. Add the new secret to `WEBHOOK_SECRETS` (e.g. `old-secret,new-secret`) and redeploy.
3. Update the webhook secret in GitHub to the new secret.
4. After confirming all deliveries use the new secret, remove the old secret from `WEBHOOK_SECRETS` and redeploy.

## Example Docker Run

```bash
docker run \
  -e PUBSUB_PROJECT_ID=my-gcp-project \
  -e PUBSUB_TOPIC_ID=github-webhooks \
  -e WEBHOOK_SECRETS=my-secret \
  -e LISTEN_PORT=8080 \
  -e LOG_LEVEL=info \
  -p 8080:8080 \
  ghook2pubsub
```
