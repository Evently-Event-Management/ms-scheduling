// internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"ms-scheduling/internal/models"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/segmentio/kafka-go"

	appconfig "ms-scheduling/internal/config"
)

// Consumer holds the dependencies for the Kafka consumer.
type Consumer struct {
	Reader          *kafka.Reader
	SchedulerClient *scheduler.Client
	Config          appconfig.Config
}

func NewConsumer(cfg appconfig.Config, kafkaURL, topic string, schedulerClient *scheduler.Client) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		GroupID: "scheduler-service-group",
	})
	return &Consumer{
		Reader:          reader,
		SchedulerClient: schedulerClient,
		Config:          cfg,
	}
}

func (c *Consumer) ConsumeDebeziumEvents() {
	for {
		msg, err := c.Reader.ReadMessage(context.Background())
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

		c.processSessionChange(event)
	}
}

// processSessionChange is the main router for Debezium operations.
func (c *Consumer) processSessionChange(event models.DebeziumEvent) {
	sessionID := ""
	if event.Payload.After != nil {
		sessionID = event.Payload.After.ID
	} else if event.Payload.Before != nil {
		sessionID = event.Payload.Before.ID // For delete operations
	}

	if sessionID == "" {
		log.Println("Could not determine session ID from Debezium event. Skipping.")
		return
	}

	log.Printf("Processing operation '%s' for session ID: %s", event.Payload.Op, sessionID)

	switch event.Payload.Op {
	case "c": // A new session was created
		log.Println("Handling create operation...")
		after := event.Payload.After
		// Schedule the on-sale job
		if after.SalesStartTime > 0 {
			onSaleTime := microsecondsToTime(after.SalesStartTime)
			err := c.createOrUpdateSchedule(after.ID, onSaleTime, "session-onsale-", c.Config.SQSONSaleQueueARN, "ON_SALE", "on-sale job")
			if err != nil {
				log.Printf("Error scheduling on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Schedule the session-closed job
		if after.EndTime > 0 {
			closedTime := microsecondsToTime(after.EndTime)
			err := c.createOrUpdateSchedule(after.ID, closedTime, "session-closed-", c.Config.SQSClosedQueueARN, "CLOSED", "closed job")
			if err != nil {
				log.Printf("Error scheduling closed job for session %s: %v", after.ID, err)
			}
		}

	case "u": // A session was updated
		log.Println("Handling update operation...")
		before, after := event.Payload.Before, event.Payload.After
		log.Printf("Before: %+v", before)
		log.Printf("After: %+v", after)
		// Sanity check
		if before == nil || after == nil {
			return
		}

		// If status changed to CANCELLED, delete schedules
		if after.Status == "CANCELLED" && before.Status != "CANCELLED" {
			log.Printf("Session %s was cancelled. Deleting schedules.", after.ID)
			c.deleteSchedule(after.ID, "session-onsale-")
			c.deleteSchedule(after.ID, "session-closed-")
			return
		}

		// Check if on-sale time changed
		if after.SalesStartTime != before.SalesStartTime {
			onSaleTime := microsecondsToTime(after.SalesStartTime)
			log.Printf("Sales start time for session %s changed. Updating schedule.", after.ID)
			c.createOrUpdateSchedule(after.ID, onSaleTime, "session-onsale-", c.Config.SQSONSaleQueueARN, "ON_SALE", "on-sale job")
		}
		// Check if end time changed
		if after.EndTime != before.EndTime {
			closedTime := microsecondsToTime(after.EndTime)
			log.Printf("End time for session %s changed. Updating schedule.", after.ID)
			c.createOrUpdateSchedule(after.ID, closedTime, "session-closed-", c.Config.SQSClosedQueueARN, "CLOSED", "closed job")
		}

	case "d": // A session was deleted
		log.Println("Handling delete operation...")
		before := event.Payload.Before
		if before == nil {
			return
		}
		c.deleteSchedule(before.ID, "session-onsale-")
		c.deleteSchedule(before.ID, "session-closed-")
	}
}

// createOrUpdateSchedule handles the idempotent logic for creating/updating a schedule.
func (c *Consumer) createOrUpdateSchedule(sessionID string, scheduleTime time.Time, namePrefix, queueArn, action, logContext string) error {
	scheduleName := namePrefix + sessionID
	log.Printf("Creating/updating schedule '%s' at time: %s", scheduleName, scheduleTime)

	// Format time for EventBridge Scheduler expression: at(YYYY-MM-DDTHH:mm:ss)
	scheduleExpression := fmt.Sprintf("at(%s)", scheduleTime.UTC().Format("2006-01-02T15:04:05"))
	inputJSON := fmt.Sprintf(`{"sessionId":"%s", "action":"%s"}`, sessionID, action)

	target := types.Target{
		Arn:     aws.String(queueArn),
		RoleArn: aws.String(c.Config.SchedulerRoleARN),
		Input:   aws.String(inputJSON),
	}

	// First, try to create the schedule
	_, err := c.SchedulerClient.CreateSchedule(context.TODO(), &scheduler.CreateScheduleInput{
		Name:                       aws.String(scheduleName),
		GroupName:                  aws.String(c.Config.SchedulerGroupName),
		ScheduleExpression:         aws.String(scheduleExpression),
		Target:                     &target,
		FlexibleTimeWindow:         &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff},
		ActionAfterCompletion:      types.ActionAfterCompletionDelete,
		ScheduleExpressionTimezone: aws.String("UTC"),
	})

	if err != nil {
		var conflict *types.ConflictException
		if errors.As(err, &conflict) {
			log.Printf("Schedule '%s' already exists. Attempting to update.", scheduleName)
			_, updateErr := c.SchedulerClient.UpdateSchedule(context.TODO(), &scheduler.UpdateScheduleInput{
				Name:                       aws.String(scheduleName),
				GroupName:                  aws.String(c.Config.SchedulerGroupName),
				ScheduleExpression:         aws.String(scheduleExpression),
				Target:                     &target,
				FlexibleTimeWindow:         &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff},
				ActionAfterCompletion:      types.ActionAfterCompletionDelete,
				ScheduleExpressionTimezone: aws.String("UTC"),
			})
			if updateErr != nil {
				log.Printf("Failed to update EventBridge schedule for %s: %v", logContext, updateErr)
				return updateErr
			}
			log.Printf("Successfully updated EventBridge schedule for %s.", logContext)
			return nil
		}
		// It was a different error
		log.Printf("Failed to create EventBridge schedule for %s: %v", logContext, err)
		return err
	}

	log.Printf("Successfully created EventBridge schedule for %s.", logContext)
	return nil
}

// deleteSchedule removes a schedule from EventBridge.
func (c *Consumer) deleteSchedule(sessionID, namePrefix string) {
	scheduleName := namePrefix + sessionID
	log.Printf("Deleting schedule '%s'", scheduleName)

	_, err := c.SchedulerClient.DeleteSchedule(context.TODO(), &scheduler.DeleteScheduleInput{
		Name:      aws.String(scheduleName),
		GroupName: aws.String(c.Config.SchedulerGroupName),
	})

	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			// This is not an error, the schedule might have already run and deleted itself.
			log.Printf("Schedule '%s' not found for deletion, it may have already completed.", scheduleName)
			return
		}
		log.Printf("Error deleting schedule '%s': %v", scheduleName, err)
	} else {
		log.Printf("Successfully deleted schedule '%s'", scheduleName)
	}
}

// microsecondsToTime converts a Debezium microsecond timestamp to a Go time.Time object.
func microsecondsToTime(microseconds int64) time.Time {
	// Convert microseconds to nanoseconds for time.Unix
	return time.Unix(0, microseconds*1000)
}
