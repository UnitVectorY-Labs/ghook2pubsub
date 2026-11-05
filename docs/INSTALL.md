---
layout: default
title: Installation
nav_order: 2
permalink: /install
---

# Installation

The project ID and topic name in these commands are placeholders. Replace them with your own values.

## Prerequisites

- A Google Cloud project with the Pub/Sub API enabled and an existing topic.
- A runtime identity with permission to publish to that topic, such as `roles/pubsub.publisher` scoped to the topic.
- Docker, or the Go version required by [go.mod](../go.mod) to build from source.
- A reachable HTTPS endpoint for GitHub deliveries. The application serves HTTP; configure TLS at your hosting platform or reverse proxy.

Create the topic if needed using an administrative identity:

```bash
gcloud pubsub topics create github-webhooks --project=my-gcp-project
```

Create downstream subscriptions before sending webhooks so those subscriptions can receive the messages.

## Build and run with Docker

From the repository root:

```bash
docker build -t ghook2pubsub .
docker run --rm -p 8080:8080 \
  -e PUBSUB_PROJECT_ID=my-gcp-project \
  -e PUBSUB_TOPIC_ID=github-webhooks \
  -e WEBHOOK_SECRETS \
  ghook2pubsub
```

Set `WEBHOOK_SECRETS` in your shell to a strong secret before running the command. Configure Application Default Credentials for the container as described below; the command alone does not provide credentials.

On Google Cloud, attach a service account with topic publishing permission to the workload. For local development, create Application Default Credentials with `gcloud auth application-default login`. A host credential file must be mounted into Docker explicitly, for example by adding:

```bash
--mount type=bind,src="$HOME/.config/gcloud/application_default_credentials.json",dst=/credentials/adc.json,readonly \
-e GOOGLE_APPLICATION_CREDENTIALS=/credentials/adc.json
```

Add these options before the image name. Ensure the container's non-root user can read the mounted file. Use your actual ADC file location if it differs.

## Build from source

```bash
go build -o ghook2pubsub .
export PUBSUB_PROJECT_ID=my-gcp-project
export PUBSUB_TOPIC_ID=github-webhooks
# Set WEBHOOK_SECRETS securely in the shell before starting.
./ghook2pubsub
```

The executable uses Application Default Credentials from its environment. See [configuration](CONFIGURATION.md) for all settings.

## Check the server

```bash
curl --fail http://localhost:8080/healthz
```

Expected response: `{"status":"ok"}`. This checks the HTTP server only; it does not test topic access or publishing. Complete a real delivery using [Usage](USAGE.md).
