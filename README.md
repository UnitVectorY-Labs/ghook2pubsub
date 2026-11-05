[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Work In Progress](https://img.shields.io/badge/Status-Work%20In%20Progress-yellow)](https://guide.unitvectorylabs.com/bestpractices/status/#work-in-progress) 

# ghook2pubsub

Consumes GitHub webhooks and publishes them to a GCP Pub/Sub topic.

The published payload can optionally be compressed by setting `PAYLOAD_COMPRESSION` to `gzip:<level>` or `zstd:<level>`. If `PAYLOAD_COMPRESSION_ATTRIBUTE` is also set, the published Pub/Sub message includes that attribute with the algorithm name only, such as `gzip` or `zstd`.

## Documentation

See the [documentation index](docs/README.md), starting with [Installation](docs/INSTALL.md) and [Usage](docs/USAGE.md). [Configuration](docs/CONFIGURATION.md), [Examples](docs/EXAMPLES.md), and the [message attribute reference](docs/ATTRIBUTES.md) cover deployment and integration.

## Related Projects

- [ghook2pubsub](https://github.com/UnitVectorY-Labs/ghook2pubsub) accepts signed GitHub webhooks and publishes them to Google Cloud Pub/Sub.
- [ghook4action](https://github.com/UnitVectorY-Labs/ghook4action) consumes opened issue and pull request events to add matching items to GitHub organization Projects.
- [pubsub2postgresaudit](https://github.com/UnitVectorY-Labs/pubsub2postgresaudit) consumes Pub/Sub messages and stores an audit history in PostgreSQL.

Use separate subscriptions on the same topic to dispatch workflows and retain an audit history independently.

## License

This project is licensed under the [MIT License](LICENSE).
