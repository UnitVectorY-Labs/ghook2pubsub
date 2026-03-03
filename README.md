# ghook2pubsub

Consumes GitHub webhooks and publishes them to a GCP Pub/Sub topic.

The published payload can optionally be compressed by setting `PAYLOAD_COMPRESSION` to `gzip:<level>` or `zstd:<level>`. If `PAYLOAD_COMPRESSION_ATTRIBUTE` is also set, the published Pub/Sub message includes that attribute with the algorithm name only, such as `gzip` or `zstd`.

## Documentation

Full documentation is available in the [`docs/`](docs/) folder:

- [Configuration](docs/configuration.md) – environment variables and Docker usage
- [Pub/Sub Attributes](docs/attributes.md) – message attribute contract
- [Architecture](docs/architecture.md) – design, request flow, and error handling

## Getting Started

Run with Docker:

```bash
docker run \
  -e PUBSUB_PROJECT_ID=my-gcp-project \
  -e PUBSUB_TOPIC_ID=github-webhooks \
  -e WEBHOOK_SECRETS=my-secret \
  -e PAYLOAD_COMPRESSION=gzip:6 \
  -e PAYLOAD_COMPRESSION_ATTRIBUTE=compression \
  -p 8080:8080 \
  ghook2pubsub
```

Then configure your GitHub webhook to point at `http://<host>:8080/webhook`.

## License

This project is licensed under the [MIT License](LICENSE).
