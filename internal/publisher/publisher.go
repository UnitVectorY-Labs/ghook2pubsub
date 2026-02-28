package publisher

import (
	"context"

	"cloud.google.com/go/pubsub"
)

// Publisher defines the interface for publishing messages.
type Publisher interface {
	Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error)
	Close() error
}

// PubSubPublisher implements Publisher using Google Cloud Pub/Sub.
type PubSubPublisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// NewPubSubPublisher creates a new PubSubPublisher for the given project and topic.
func NewPubSubPublisher(ctx context.Context, projectID, topicID string) (*PubSubPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicID)

	return &PubSubPublisher{
		client: client,
		topic:  topic,
	}, nil
}

// Publish sends data with attributes to Pub/Sub and returns the server-assigned message ID.
func (p *PubSubPublisher) Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error) {
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	})

	serverID, err := result.Get(ctx)
	if err != nil {
		return "", err
	}

	return serverID, nil
}

// Close stops the topic and closes the underlying client.
func (p *PubSubPublisher) Close() error {
	p.topic.Stop()
	return p.client.Close()
}
