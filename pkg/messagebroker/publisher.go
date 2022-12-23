package event

import (
	"context"
	"errors"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Publisher ...
type Publisher struct {
	topic            string
	cloudEventClient cloudevents.Client
}

// AddPublisher ...
func (kafka *Kafka) AddPublisher(topic string) {
	if kafka.publishers[topic] != nil {
		return
	}

	sender, err := kafka_sarama.NewSender(
		[]string{kafka.cfg.KafkaUrl}, // Kafka connection url
		kafka.saramaConfig,           // Kafka sarama config
		topic,                        // Topic
	)

	if err != nil {
		panic(err)
	}

	c, err := cloudevents.NewClient(sender, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		panic(err)
	}

	kafka.publishers[topic] = &Publisher{
		topic:            topic,
		cloudEventClient: c,
	}
}

// Push ...
func (r *Kafka) Push(topic string, e cloudevents.Event) error {
	p := r.publishers[topic]
	if p == nil {
		return errors.New("publisher with that topic doesn't exists: " + topic)
	}

	result := p.cloudEventClient.Send(
		kafka_sarama.WithMessageKey(context.Background(), sarama.StringEncoder(e.ID())),
		e,
	)

	if cloudevents.IsUndelivered(result) {
		return errors.New("failed to publish event")
	}

	return nil
}
