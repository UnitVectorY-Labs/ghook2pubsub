# ghook2pubsub Documentation

Accept signed GitHub webhooks and publish their original body to Google Cloud Pub/Sub, with attributes for downstream routing. Optional compression changes the message bytes, not the webhook content.

## Guides

- [Installation](INSTALL.md): prepare Pub/Sub, credentials, and run the service.
- [Configuration](CONFIGURATION.md): environment variables, compression, and secret rotation.
- [Usage](USAGE.md): connect GitHub, verify deliveries, and troubleshoot failures.
- [Examples](EXAMPLES.md): inspect a delivery and configure downstream consumers.
- [Message attributes](ATTRIBUTES.md): attribute names and extraction behavior.

Start with installation, then configure a GitHub webhook using the usage guide.
