package event

import (
	"github.com/TemurMannonov/medium_user_service/config"

	"github.com/Shopify/sarama"
)

type Kafka struct {
	cfg          config.Config
	saramaConfig *sarama.Config
	publishers   map[string]*Publisher
}

func NewKafka(cfg config.Config) *Kafka {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_0_0_0

	kafka := &Kafka{
		cfg:          cfg,
		saramaConfig: saramaConfig,
		publishers:   make(map[string]*Publisher),
	}

	kafka.AddPublisher("v1.notification_service.send_email")

	return kafka
}
