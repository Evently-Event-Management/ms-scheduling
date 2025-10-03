package kafka

import (
	"context"
	"encoding/json"
	"log"
	"ms-scheduling/internal/models"

	"github.com/segmentio/kafka-go"
)

func NewConsumer(kafkaURL, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		GroupID: "scheduler-service-group",
	})
}

func ConsumeDebeziumEvents(consumer *kafka.Reader) {
	for {
		msg, err := consumer.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading from Kafka: %v", err)
			continue
		}

		log.Printf("Received Kafka message from topic %s", msg.Topic)

		var event models.DebeziumEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshalling Debezium event: %v", err)
			continue
		}

		// At this point, 'event' is a Go struct with your data.
		// Now you can process it.
		processSessionChange(event)
	}
}

func processSessionChange(event models.DebeziumEvent) {
	// This is where you'll add the scheduling logic
	log.Printf("Processing operation '%s' for session ID: %s", event.Payload.Op, event.Payload.After.ID)

	switch event.Payload.Op {
	case "c": // Create
		// A new session was created
		// Call your function to create EventBridge schedules
		// scheduleOnSaleJob(event.Payload.After)
		// scheduleClosedJob(event.Payload.After)
		log.Println("Handling create operation...")
	case "u": // Update
		// A session was updated
		// Compare event.Payload.Before and event.Payload.After
		// to see if startTime or endTime changed, then update schedules.
		log.Println("Handling update operation...")
	case "d": // Delete
		// A session was deleted
		// Call your function to delete the EventBridge schedules
		log.Println("Handling delete operation...")
	}
}
