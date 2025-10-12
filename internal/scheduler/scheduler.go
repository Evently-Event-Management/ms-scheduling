// internal/scheduler/scheduler.go
package scheduler

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

	appconfig "ms-scheduling/internal/config"
)

// Service encapsulates the EventBridge Scheduler functionality.
type Service struct {
	SchedulerClient *scheduler.Client
	Config          appconfig.Config
}

// NewService creates a new scheduler service.
func NewService(cfg appconfig.Config, schedulerClient *scheduler.Client) *Service {
	return &Service{
		SchedulerClient: schedulerClient,
		Config:          cfg,
	}
}

// CreateOrUpdateSchedule handles the idempotent logic for creating/updating a schedule.
func (s *Service) CreateOrUpdateSchedule(sessionID string, scheduleTime time.Time, namePrefix, queueArn, action, logContext string) error {
	scheduleName := namePrefix + sessionID
	log.Printf("Creating/updating schedule '%s' at time: %s", scheduleName, scheduleTime)

	// Format time for EventBridge Scheduler expression: at(YYYY-MM-DDTHH:mm:ss)
	scheduleExpression := fmt.Sprintf("at(%s)", scheduleTime.UTC().Format("2006-01-02T15:04:05"))

	// Use the SQSMessageBody struct to ensure consistency between producer and consumer
	messageBody := models.SQSMessageBody{
		SessionID: sessionID,
		Action:    action,
	}

	// Marshal the struct to JSON
	inputJSON, err := json.Marshal(messageBody)
	if err != nil {
		log.Printf("Error marshaling message body to JSON: %v", err)
		return err
	}

	target := types.Target{
		Arn:     aws.String(queueArn),
		RoleArn: aws.String(s.Config.SchedulerRoleARN),
		Input:   aws.String(string(inputJSON)),
	}

	// First, try to create the schedule
	_, err = s.SchedulerClient.CreateSchedule(context.TODO(), &scheduler.CreateScheduleInput{
		Name:                       aws.String(scheduleName),
		GroupName:                  aws.String(s.Config.SchedulerGroupName),
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
			_, updateErr := s.SchedulerClient.UpdateSchedule(context.TODO(), &scheduler.UpdateScheduleInput{
				Name:                       aws.String(scheduleName),
				GroupName:                  aws.String(s.Config.SchedulerGroupName),
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

// DeleteSchedule removes a schedule from EventBridge.
func (s *Service) DeleteSchedule(sessionID, namePrefix string) {
	scheduleName := namePrefix + sessionID
	log.Printf("Deleting schedule '%s'", scheduleName)

	_, err := s.SchedulerClient.DeleteSchedule(context.TODO(), &scheduler.DeleteScheduleInput{
		Name:      aws.String(scheduleName),
		GroupName: aws.String(s.Config.SchedulerGroupName),
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

// MicrosecondsToTime converts a Debezium microsecond timestamp to a Go time.Time object.
func MicrosecondsToTime(microseconds int64) time.Time {
	return time.Unix(0, microseconds*1000)
}
