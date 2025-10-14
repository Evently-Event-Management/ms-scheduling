package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"

	"ms-scheduling/internal/config"
)

// BaseConsumer provides common functionality for all Kafka consumers
type BaseConsumer struct {
	Reader *kafka.Reader
	Config config.Config
}

// NewBaseConsumer creates a new base consumer with the given configuration
func NewBaseConsumer(cfg config.Config, kafkaURL, topic string) *BaseConsumer {
	// If topic is empty, return a consumer with nil reader
	if topic == "" || kafkaURL == "" {
		log.Println("Empty Kafka topic or URL provided, skipping consumer creation")
		return &BaseConsumer{
			Reader: nil,
			Config: cfg,
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		GroupID: "scheduler-service-group",
	})

	return &BaseConsumer{
		Reader: reader,
		Config: cfg,
	}
}

// Close closes the Kafka reader
func (c *BaseConsumer) Close() error {
	return c.Reader.Close()
}

// ConsumeMessages consumes messages from Kafka and passes them to the provided handler function
func (c *BaseConsumer) ConsumeMessages(ctx context.Context, handler func([]byte) error) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer")
			return
		default:
			msg, err := c.Reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading from Kafka: %v", err)
				continue
			}

			log.Printf("Received Kafka message from topic %s", msg.Topic)

			if err := handler(msg.Value); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}
}
