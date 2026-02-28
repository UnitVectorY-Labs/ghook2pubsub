# ghook2pubsub

Consumes GitHub webhooks and publishes them to a GCP Pub/Sub topic.

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
  -p 8080:8080 \
  ghook2pubsub
```

Then configure your GitHub webhook to point at `http://<host>:8080/webhook`.

## License

This project is licensed under the [MIT License](LICENSE).
