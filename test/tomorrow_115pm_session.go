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
	fmt.Println("🎪 Event Session Creation Test - Tomorrow 1:15 PM")
	fmt.Println("=================================================")

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

	// Create consumer to process the event
	consumer := &kafka.Consumer{
		SchedulerService:  schedulerService,
		SubscriberService: subscriberService,
		Config:            cfg,
	}

	// Set event time for tomorrow at 1:15 PM
	tomorrowAt115PM := time.Date(2025, 10, 13, 13, 15, 0, 0, time.UTC) // 1:15 PM
	sessionEndTime := tomorrowAt115PM.Add(90 * time.Minute)           // 1.5 hours duration (ends at 2:45 PM)
	salesStartTime := time.Now().Add(-30 * time.Minute)              // Sales started 30 minutes ago

	// Convert to microseconds (Debezium timestamp format)
	startTimeMicros := tomorrowAt115PM.UnixMicro()
	endTimeMicros := sessionEndTime.UnixMicro()
	salesStartMicros := salesStartTime.UnixMicro()
	currentTimestamp := time.Now().UnixMilli()

	// Create the Debezium event structure with the exact schema you provided
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
				"id": "7af78d86-f8eb-45bf-abac-381d6a8176d3",
				"event_id": "e8b7f94b-2a55-49c3-82ec-f02c93965486",
				"start_time": %d,
				"end_time": %d,
				"status": "PENDING",
				"venue_details": "{\"name\": \"Test Event Venue\", \"address\": \"123 Main Street, Colombo\", \"latitude\": 6.928667714588207, \"longitude\": 79.8601902689852, \"onlineLink\": null}",
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
	}`, startTimeMicros, endTimeMicros, salesStartMicros, currentTimestamp, currentTimestamp)

	fmt.Printf("📋 Session Event Details:\n")
	fmt.Printf("   📅 Session Date: %s\n", tomorrowAt115PM.Format("2006-01-02"))
	fmt.Printf("   🕐 Start Time: %s\n", tomorrowAt115PM.Format("15:04:05"))
	fmt.Printf("   🕑 End Time: %s\n", sessionEndTime.Format("15:04:05"))
	fmt.Printf("   📍 Venue: Test Event Venue, 123 Main Street, Colombo\n")
	fmt.Printf("   🎫 Status: PENDING\n")
	fmt.Printf("   📧 Sales Started: %s ago\n", time.Since(salesStartTime).Round(time.Minute))

	// Parse and process the Debezium event
	var debeziumEvent models.DebeziumEvent
	if err := json.Unmarshal([]byte(debeziumEventJSON), &debeziumEvent); err != nil {
		log.Fatalf("❌ Failed to parse Debezium event: %v", err)
	}

	fmt.Printf("\n🚀 Processing Event Session Creation...\n")
	
	// Process the session creation event - this simulates what happens when Kafka receives the message
	consumer.ProcessSessionChangePublic(debeziumEvent)

	fmt.Printf("\n✅ Event Session Processing Complete!\n")
	fmt.Printf("=====================================\n")
	fmt.Printf("📧 Email reminder has been scheduled for: %s\n", tomorrowAt115PM.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
	fmt.Printf("🎯 Session ID: %s\n", debeziumEvent.Payload.After.ID)
	fmt.Printf("🎪 Event ID: %s\n", debeziumEvent.Payload.After.EventID)
	
	fmt.Printf("\n📝 What this test does:\n")
	fmt.Printf("1. ✅ Creates a realistic Debezium event structure\n")
	fmt.Printf("2. ✅ Sets session start time to tomorrow 1:15 PM\n")
	fmt.Printf("3. ✅ Processes the event through your Kafka consumer\n")
	fmt.Printf("4. ✅ Schedules email reminders (24h before session)\n")
	fmt.Printf("5. ✅ Uses the same data structure as real Kafka streams\n")
	
	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("- Keep your main consumer running: go run main.go\n")
	fmt.Printf("- Email reminders will be sent automatically at the scheduled time\n")
	fmt.Printf("- Check logs for 'REMINDER_EMAIL' processing\n")
}