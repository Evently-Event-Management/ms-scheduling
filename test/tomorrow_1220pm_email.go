package main

import (
	"encoding/json"
	"fmt"
	"log"
	"ms-scheduling/internal/kafka"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/scheduler"
	"ms-scheduling/internal/services"
	"time"

	appconfig "ms-scheduling/internal/config"
)

func main() {
	fmt.Println("🔔 Session Reminder Email Test - Tomorrow 12:20 PM")
	fmt.Println("==================================================")

	// Load configuration
	appconfig.LoadEnv()
	cfg := appconfig.Config{
		AWSRegion:                    "ap-south-1",
		SQSSessionSchedulingQueueARN: "arn:aws:sqs:ap-south-1:621014405736:session-scheduling-queue-infra-dev-isurumuni",
		SchedulerRoleARN:            "arn:aws:iam::621014405736:role/EventBridgeSchedulerSqsRole",
		SchedulerGroupName:          "event-ticketing-schedules-infra-dev-isurumuni",
		DatabaseHost:                "localhost",
		DatabasePort:                "5432",
		DatabaseUser:                "ticketly",
		DatabasePassword:            "ticketly",
		DatabaseName:                "ms_scheduling",
		SMTPHost:                    "smtp.gmail.com",
		SMTPPort:                    "587",
		SMTPUsername:                "isurumuniwije@gmail.com",
		SMTPPassword:                "yotp eehv mcnq osnh",
		FromEmail:                   "noreply@ticketly.com",
		FromName:                    "Ticketly Support",
	}

	// Initialize services
	schedulerService, err := scheduler.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create scheduler service: %v", err)
	}

	subscriberService, err := services.NewSubscriberService(cfg)
	if err != nil {
		log.Fatalf("Failed to create subscriber service: %v", err)
	}

	// Create consumer (we'll use it to process the event)
	consumer := &kafka.Consumer{
		SchedulerService:  schedulerService,
		SubscriberService: subscriberService,
		Config:            cfg,
	}

	// Create the exact Debezium event structure you provided, but with tomorrow 12:20 PM start time
	tomorrowAt1220PM := time.Date(2025, 10, 13, 12, 20, 0, 0, time.UTC)
	sessionEndTime := tomorrowAt1220PM.Add(2 * time.Hour) // 2 hours duration
	salesStartTime := time.Now().Add(-1 * time.Hour)      // Sales started 1 hour ago

	// Convert to microseconds (Debezium format)
	startTimeMicros := tomorrowAt1220PM.UnixMicro()
	endTimeMicros := sessionEndTime.UnixMicro()
	salesStartMicros := salesStartTime.UnixMicro()

	debeziumEventJSON := fmt.Sprintf(`{
		"schema": {
			"type": "struct",
			"fields": [
				{
					"type": "struct",
					"fields": [
						{
							"type": "string",
							"optional": false,
							"name": "io.debezium.data.Uuid",
							"version": 1,
							"field": "id"
						},
						{
							"type": "string",
							"optional": false,
							"name": "io.debezium.data.Uuid",
							"version": 1,
							"field": "event_id"
						},
						{
							"type": "int64",
							"optional": false,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "start_time"
						},
						{
							"type": "int64",
							"optional": false,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "end_time"
						},
						{
							"type": "string",
							"optional": false,
							"field": "status"
						},
						{
							"type": "string",
							"optional": true,
							"name": "io.debezium.data.Json",
							"version": 1,
							"field": "venue_details"
						},
						{
							"type": "string",
							"optional": false,
							"field": "session_type"
						},
						{
							"type": "int64",
							"optional": true,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "sales_start_time"
						}
					],
					"optional": true,
					"name": "dbz.ticketly.public.event_sessions.Value",
					"field": "before"
				},
				{
					"type": "struct",
					"fields": [
						{
							"type": "string",
							"optional": false,
							"name": "io.debezium.data.Uuid",
							"version": 1,
							"field": "id"
						},
						{
							"type": "string",
							"optional": false,
							"name": "io.debezium.data.Uuid",
							"version": 1,
							"field": "event_id"
						},
						{
							"type": "int64",
							"optional": false,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "start_time"
						},
						{
							"type": "int64",
							"optional": false,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "end_time"
						},
						{
							"type": "string",
							"optional": false,
							"field": "status"
						},
						{
							"type": "string",
							"optional": true,
							"name": "io.debezium.data.Json",
							"version": 1,
							"field": "venue_details"
						},
						{
							"type": "string",
							"optional": false,
							"field": "session_type"
						},
						{
							"type": "int64",
							"optional": true,
							"name": "io.debezium.time.MicroTimestamp",
							"version": 1,
							"field": "sales_start_time"
						}
					],
					"optional": true,
					"name": "dbz.ticketly.public.event_sessions.Value",
					"field": "after"
				},
				{
					"type": "struct",
					"fields": [
						{
							"type": "string",
							"optional": false,
							"field": "version"
						},
						{
							"type": "string",
							"optional": false,
							"field": "connector"
						},
						{
							"type": "string",
							"optional": false,
							"field": "name"
						},
						{
							"type": "int64",
							"optional": false,
							"field": "ts_ms"
						},
						{
							"type": "string",
							"optional": true,
							"name": "io.debezium.data.Enum",
							"version": 1,
							"parameters": {
								"allowed": "true,last,false,incremental"
							},
							"default": "false",
							"field": "snapshot"
						},
						{
							"type": "string",
							"optional": false,
							"field": "db"
						},
						{
							"type": "string",
							"optional": true,
							"field": "sequence"
						},
						{
							"type": "string",
							"optional": false,
							"field": "schema"
						},
						{
							"type": "string",
							"optional": false,
							"field": "table"
						},
						{
							"type": "int64",
							"optional": true,
							"field": "txId"
						},
						{
							"type": "int64",
							"optional": true,
							"field": "lsn"
						},
						{
							"type": "int64",
							"optional": true,
							"field": "xmin"
						}
					],
					"optional": false,
					"name": "io.debezium.connector.postgresql.Source",
					"field": "source"
				},
				{
					"type": "string",
					"optional": false,
					"field": "op"
				},
				{
					"type": "int64",
					"optional": true,
					"field": "ts_ms"
				},
				{
					"type": "struct",
					"fields": [
						{
							"type": "string",
							"optional": false,
							"field": "id"
						},
						{
							"type": "int64",
							"optional": false,
							"field": "total_order"
						},
						{
							"type": "int64",
							"optional": false,
							"field": "data_collection_order"
						}
					],
					"optional": true,
					"name": "event.block",
					"version": 1,
					"field": "transaction"
				}
			],
			"optional": false,
			"name": "dbz.ticketly.public.event_sessions.Envelope",
			"version": 1
		},
		"payload": {
			"before": null,
			"after": {
				"id": "999998",
				"event_id": "999999",
				"start_time": %d,
				"end_time": %d,
				"status": "ON_SALE",
				"venue_details": "{\"name\": \"Tomorrow Test Venue\", \"address\": \"123 Test Avenue, Tomorrow City\", \"latitude\": 6.928667714588207, \"longitude\": 79.8601902689852, \"onlineLink\": null}",
				"session_type": "PHYSICAL",
				"sales_start_time": %d
			},
			"source": {
				"version": "2.5.4.Final",
				"connector": "postgresql",
				"name": "dbz.ticketly",
				"ts_ms": %d,
				"snapshot": "false",
				"db": "event_service",
				"sequence": "[\"50893648\",\"50897664\"]",
				"schema": "public",
				"table": "event_sessions",
				"txId": 1219,
				"lsn": 50897664,
				"xmin": null
			},
			"op": "c",
			"ts_ms": %d,
			"transaction": null
		}
	}`, startTimeMicros, endTimeMicros, salesStartMicros, time.Now().UnixMilli(), time.Now().UnixMilli())

	// Parse the JSON into Debezium event
	var debeziumEvent models.DebeziumEvent
	if err := json.Unmarshal([]byte(debeziumEventJSON), &debeziumEvent); err != nil {
		log.Fatalf("Failed to unmarshal Debezium event: %v", err)
	}

	// Display test information
	fmt.Printf("📅 Test Session Details:\n")
	fmt.Printf("   Session ID: %s\n", debeziumEvent.Payload.After.ID)
	fmt.Printf("   Event ID: %s\n", debeziumEvent.Payload.After.EventID)
	fmt.Printf("   Session Start: %s\n", tomorrowAt1220PM.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Session End: %s\n", sessionEndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Status: %s\n", debeziumEvent.Payload.After.Status)
	fmt.Printf("   Type: %s\n", debeziumEvent.Payload.After.SessionType)

	// Calculate reminder time (24 hours before session start)
	reminderTime := tomorrowAt1220PM.AddDate(0, 0, -1)
	
	fmt.Printf("\n⏰ Reminder Schedule:\n")
	fmt.Printf("   Reminder Time: %s (24 hours before session)\n", reminderTime.Format("2006-01-02 15:04:05"))
	
	now := time.Now()
	if reminderTime.After(now) {
		timeTilReminder := reminderTime.Sub(now)
		fmt.Printf("   Time until reminder: %s\n", timeTilReminder.String())
		fmt.Printf("   Status: ✅ Reminder will be sent automatically\n")
	} else {
		fmt.Printf("   Status: ⚠️ Reminder time has passed\n")
	}

	fmt.Printf("\n🎯 Processing Debezium Event (Create Operation):\n")
	
	// Process the session change - this will schedule the reminder email
	consumer.ProcessSessionChangePublic(debeziumEvent)

	fmt.Printf("\n📧 Email Reminder Setup Complete!\n")
	fmt.Printf("=======================================\n")
	fmt.Printf("✅ Session reminder scheduled for: %s\n", reminderTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("📬 Email will be sent to all subscribers of:\n")
	fmt.Printf("   - Event: %s\n", debeziumEvent.Payload.After.EventID)
	fmt.Printf("   - Session: %s\n", debeziumEvent.Payload.After.ID)
	fmt.Printf("\n🔍 What happens next:\n")
	fmt.Printf("1. AWS EventBridge scheduler is set for the reminder time\n")
	fmt.Printf("2. At the scheduled time, SQS message will be sent with action 'REMINDER_EMAIL'\n")
	fmt.Printf("3. Your Kafka consumer will process the SQS message\n")
	fmt.Printf("4. Session reminder emails will be sent to subscribers\n")
	fmt.Printf("5. Check email: isurumuni.22@cse.mrt.ac.lk\n")
	
	fmt.Printf("\n🚀 To see the system in action:\n")
	fmt.Printf("1. Keep your Kafka consumer running: go run main.go\n")
	fmt.Printf("2. Wait for the reminder time: %s\n", reminderTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("3. Watch for 'REMINDER_EMAIL' processing in the logs\n")
	fmt.Printf("4. Check your email inbox for the session reminder\n")
}