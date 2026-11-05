# Examples

## Inspect a delivery

Create a temporary subscription before sending a test GitHub event:

```bash
gcloud pubsub subscriptions create github-webhooks-inspect \
  --topic=github-webhooks --project=my-gcp-project
```

Trigger an event or redeliver an existing GitHub delivery, then pull the message:

```bash
gcloud pubsub subscriptions pull github-webhooks-inspect \
  --auto-ack --limit=1 --format=json --project=my-gcp-project
```

The output includes the message data and attributes such as `gh_event`, `gh_delivery`, and `repository` when supplied by the event. JSON output represents message data as base64; decode it before inspecting the original body. With compression enabled, decompress the decoded bytes too.

Remove the temporary subscription afterward:

```bash
gcloud pubsub subscriptions delete github-webhooks-inspect --project=my-gcp-project
```

## Route events to separate consumers

Create a subscription that receives only push events:

```bash
gcloud pubsub subscriptions create github-push-events \
  --topic=github-webhooks --project=my-gcp-project \
  --message-filter='attributes.gh_event = "push"'
```

Use separate subscriptions on the same topic for independent consumers: [ghook4action](https://github.com/UnitVectorY-Labs/ghook4action) can dispatch GitHub Actions workflows, while [pubsub2postgresaudit](https://github.com/UnitVectorY-Labs/pubsub2postgresaudit) can store an audit history in PostgreSQL. Configure each consumer using its own documentation.

## Compress payloads

Set these variables on the publishing service:

```bash
export PAYLOAD_COMPRESSION=gzip:6
export PAYLOAD_COMPRESSION_ATTRIBUTE=compression
```

Published message data is now gzip compressed and the `compression` attribute is `gzip`. Configure every consumer to decode this format and attribute name before enabling it. Compression is disabled by default; subscribers receive the original request bytes in that mode. See the [attribute contract](ATTRIBUTES.md) for available routing metadata.
