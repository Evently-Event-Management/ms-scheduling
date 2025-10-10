package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

// Kafka Producer Test for Order Created Events
func main() {
	fmt.Println("🚀 Kafka Order Event Producer Test")
	fmt.Println(strings.Repeat("=", 40))

	// Kafka configuration
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	// Create producer
	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		log.Fatalf("Error creating Kafka producer: %v", err)
	}
	defer producer.Close()

	// Create test order events
	testOrders := createTestOrderEvents()

	fmt.Printf("📨 Publishing %d test order events to Kafka...\n", len(testOrders))

	for i, order := range testOrders {
		// Marshal order to JSON
		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("❌ Error marshaling order %d: %v", i+1, err)
			continue
		}

		// Create Kafka message
		message := &sarama.ProducerMessage{
			Topic: "ticketly.order.created",
			Key:   sarama.StringEncoder(order.OrderID),
			Value: sarama.StringEncoder(orderJSON),
			Headers: []sarama.RecordHeader{
				{
					Key:   []byte("eventType"),
					Value: []byte("order.created"),
				},
				{
					Key:   []byte("version"),
					Value: []byte("1.0"),
				},
			},
		}

		// Send message
		partition, offset, err := producer.SendMessage(message)
		if err != nil {
			log.Printf("❌ Error sending order %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Order %d sent successfully - Partition: %d, Offset: %d\n",
				i+1, partition, offset)
			fmt.Printf("   Order ID: %s, User: %s, Price: $%.2f\n",
				order.OrderID, order.UserID, order.Price)
		}

		// Small delay between messages
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n🎉 Kafka test completed!")
	fmt.Println("💡 Check your main service logs to see if emails were sent.")
}

// OrderCreatedEvent represents the Kafka event structure
type OrderCreatedEvent struct {
	OrderID        string   `json:"OrderID"`
	UserID         string   `json:"UserID"`
	EventID        string   `json:"EventID"`
	SessionID      string   `json:"SessionID"`
	Status         string   `json:"Status"`
	SubTotal       float64  `json:"SubTotal"`
	DiscountID     string   `json:"DiscountID"`
	DiscountCode   string   `json:"DiscountCode"`
	DiscountAmount float64  `json:"DiscountAmount"`
	Price          float64  `json:"Price"`
	CreatedAt      string   `json:"CreatedAt"`
	PaymentAT      string   `json:"PaymentAT"`
	Tickets        []Ticket `json:"tickets"`
}

type Ticket struct {
	TicketID        string  `json:"ticket_id"`
	OrderID         string  `json:"order_id"`
	SeatID          string  `json:"seat_id"`
	SeatLabel       string  `json:"seat_label"`
	Colour          string  `json:"colour"`
	TierID          string  `json:"tier_id"`
	TierName        string  `json:"tier_name"`
	PriceAtPurchase float64 `json:"price_at_purchase"`
	IssuedAt        string  `json:"issued_at"`
	CheckedIn       bool    `json:"checked_in"`
	CheckedInTime   string  `json:"checked_in_time"`
}

// createTestOrderEvents creates multiple test order events
func createTestOrderEvents() []OrderCreatedEvent {
	baseTime := time.Now()

	return []OrderCreatedEvent{
		// Test Order 1: Single VIP ticket
		{
			OrderID:        fmt.Sprintf("ORD_VIP_%d", baseTime.Unix()),
			UserID:         "john.doe@example.com",
			EventID:        "evt_concert_taylor_swift",
			SessionID:      "sess_main_show_evening",
			Status:         "CONFIRMED",
			SubTotal:       200.00,
			DiscountID:     "",
			DiscountCode:   "",
			DiscountAmount: 0.00,
			Price:          200.00,
			CreatedAt:      baseTime.Format(time.RFC3339),
			PaymentAT:      baseTime.Add(2 * time.Minute).Format(time.RFC3339),
			Tickets: []Ticket{
				{
					TicketID:        "tkt_vip_001",
					OrderID:         fmt.Sprintf("ORD_VIP_%d", baseTime.Unix()),
					SeatID:          "VIP_A1",
					SeatLabel:       "VIP Section A, Row 1, Seat 1",
					Colour:          "#FFD700",
					TierID:          "tier_vip_platinum",
					TierName:        "VIP Platinum",
					PriceAtPurchase: 200.00,
					IssuedAt:        baseTime.Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
			},
		},

		// Test Order 2: Family package with discount
		{
			OrderID:        fmt.Sprintf("ORD_FAM_%d", baseTime.Add(1*time.Minute).Unix()),
			UserID:         "sarah.johnson@example.com",
			EventID:        "evt_concert_taylor_swift",
			SessionID:      "sess_main_show_evening",
			Status:         "CONFIRMED",
			SubTotal:       300.00,
			DiscountID:     "FAMILY_PACK",
			DiscountCode:   "FAM20",
			DiscountAmount: 60.00,
			Price:          240.00,
			CreatedAt:      baseTime.Add(1 * time.Minute).Format(time.RFC3339),
			PaymentAT:      baseTime.Add(3 * time.Minute).Format(time.RFC3339),
			Tickets: []Ticket{
				{
					TicketID:        "tkt_std_001",
					OrderID:         fmt.Sprintf("ORD_FAM_%d", baseTime.Add(1*time.Minute).Unix()),
					SeatID:          "STD_B12",
					SeatLabel:       "Standard Section B, Row 1, Seat 12",
					Colour:          "#4A90E2",
					TierID:          "tier_standard",
					TierName:        "Standard",
					PriceAtPurchase: 75.00,
					IssuedAt:        baseTime.Add(1 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
				{
					TicketID:        "tkt_std_002",
					OrderID:         fmt.Sprintf("ORD_FAM_%d", baseTime.Add(1*time.Minute).Unix()),
					SeatID:          "STD_B13",
					SeatLabel:       "Standard Section B, Row 1, Seat 13",
					Colour:          "#4A90E2",
					TierID:          "tier_standard",
					TierName:        "Standard",
					PriceAtPurchase: 75.00,
					IssuedAt:        baseTime.Add(1 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
				{
					TicketID:        "tkt_std_003",
					OrderID:         fmt.Sprintf("ORD_FAM_%d", baseTime.Add(1*time.Minute).Unix()),
					SeatID:          "STD_B14",
					SeatLabel:       "Standard Section B, Row 1, Seat 14",
					Colour:          "#4A90E2",
					TierID:          "tier_standard",
					TierName:        "Standard",
					PriceAtPurchase: 75.00,
					IssuedAt:        baseTime.Add(1 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
				{
					TicketID:        "tkt_std_004",
					OrderID:         fmt.Sprintf("ORD_FAM_%d", baseTime.Add(1*time.Minute).Unix()),
					SeatID:          "STD_B15",
					SeatLabel:       "Standard Section B, Row 1, Seat 15",
					Colour:          "#4A90E2",
					TierID:          "tier_standard",
					TierName:        "Standard",
					PriceAtPurchase: 75.00,
					IssuedAt:        baseTime.Add(1 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
			},
		},

		// Test Order 3: Budget tickets
		{
			OrderID:        fmt.Sprintf("ORD_ECO_%d", baseTime.Add(2*time.Minute).Unix()),
			UserID:         "mike.wilson@example.com",
			EventID:        "evt_concert_taylor_swift",
			SessionID:      "sess_main_show_evening",
			Status:         "CONFIRMED",
			SubTotal:       100.00,
			DiscountID:     "STUDENT",
			DiscountCode:   "STU15",
			DiscountAmount: 15.00,
			Price:          85.00,
			CreatedAt:      baseTime.Add(2 * time.Minute).Format(time.RFC3339),
			PaymentAT:      baseTime.Add(4 * time.Minute).Format(time.RFC3339),
			Tickets: []Ticket{
				{
					TicketID:        "tkt_eco_001",
					OrderID:         fmt.Sprintf("ORD_ECO_%d", baseTime.Add(2*time.Minute).Unix()),
					SeatID:          "ECO_C25",
					SeatLabel:       "Economy Section C, Row 2, Seat 25",
					Colour:          "#7ED321",
					TierID:          "tier_economy",
					TierName:        "Economy",
					PriceAtPurchase: 50.00,
					IssuedAt:        baseTime.Add(2 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
				{
					TicketID:        "tkt_eco_002",
					OrderID:         fmt.Sprintf("ORD_ECO_%d", baseTime.Add(2*time.Minute).Unix()),
					SeatID:          "ECO_C26",
					SeatLabel:       "Economy Section C, Row 2, Seat 26",
					Colour:          "#7ED321",
					TierID:          "tier_economy",
					TierName:        "Economy",
					PriceAtPurchase: 50.00,
					IssuedAt:        baseTime.Add(2 * time.Minute).Format(time.RFC3339),
					CheckedIn:       false,
					CheckedInTime:   "",
				},
			},
		},
	}
}
