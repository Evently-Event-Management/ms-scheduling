package main

import (
	"database/sql"
	"fmt"
	"log"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Simple test for order email functionality without Kafka dependency
func TestOrderEmail() {
	fmt.Println("🧪 Order Email Test - Direct Database & Service Test")
	fmt.Println(strings.Repeat("=", 60))

	// Database connection
	db, err := sql.Open("postgres",
		"host=localhost port=5432 user=ticketly password=ticketly dbname=ms_scheduling sslmode=disable")
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Database not accessible: %v", err)
	}
	fmt.Println("✅ Database connected successfully")

	// Initialize services
	keycloakClient := services.NewKeycloakClient(
		"http://auth.ticketly.com:8080",
		"event-ticketing",
		"scheduler-service-client",
		"rIyS9oGTSQmZMUXhEm4NZSKxmYd8a8jU",
	)

	emailService := services.NewEmailService(
		"smtp.gmail.com", "587", "isurumuniwije@gmail.com",
		"yotp eehv mcnq osnh", "noreply@ticketly.com", "Ticketly Support",
	)

	subscriberService := services.NewSubscriberService(db, keycloakClient, emailService)
	fmt.Println("✅ Services initialized")

	// Step 1: Create test order event
	testOrder := createTestOrder()
	fmt.Println("\n📦 Test Order Created:")
	printOrderSummary(testOrder)

	// Step 2: Create/Get subscriber (UserID is now a UUID)
	fmt.Println("\n👤 Managing Subscriber...")
	subscriber, err := createOrGetSubscriber(db, testOrder.UserID)
	if err != nil {
		log.Fatalf("❌ Failed to create subscriber: %v", err)
	}
	fmt.Printf("✅ Subscriber: ID=%d, Email=%s\n", subscriber.SubscriberID, subscriber.SubscriberMail)

	// Step 3: Add subscription for the event
	fmt.Println("\n📋 Adding Event Subscription...")
	err = addSubscription(db, subscriber.SubscriberID, "event", 1001) // Using 1001 as sample event ID
	if err != nil {
		log.Printf("⚠️ Subscription warning: %v", err)
	} else {
		fmt.Println("✅ Event subscription added")
	}

	// Step 4: Send confirmation email
	fmt.Println("\n📧 Sending Order Confirmation Email...")
	err = subscriberService.SendOrderConfirmationEmail(subscriber, testOrder)
	if err != nil {
		log.Printf("❌ Email sending failed: %v", err)
		fmt.Println("💡 Note: This might fail if SMTP settings are incorrect for testing")
	} else {
		fmt.Println("✅ Email sent successfully!")
	}

	// Step 5: Verify database records
	fmt.Println("\n🔍 Database Verification:")
	verifySubscriber(db, subscriber.SubscriberID)

	fmt.Println("\n🎉 Test Completed Successfully!")
	fmt.Println(strings.Repeat("=", 60))

	// Print summary
	fmt.Println("📊 Test Summary:")
	fmt.Printf("   - Order ID: %s\n", testOrder.OrderID)
	fmt.Printf("   - User UUID: %s\n", testOrder.UserID)
	fmt.Printf("   - Email: %s\n", subscriber.SubscriberMail)
	fmt.Printf("   - Event: %s\n", testOrder.EventID)
	fmt.Printf("   - Total: $%.2f\n", testOrder.Price)
	fmt.Printf("   - Tickets: %d\n", len(testOrder.Tickets))
	fmt.Printf("   - Subscriber ID: %d\n", subscriber.SubscriberID)
}

// createTestOrder creates a sample order for testing
func createTestOrder() *services.OrderCreatedEvent {
	timestamp := time.Now()
	orderID := fmt.Sprintf("TEST_ORDER_%d", timestamp.Unix())

	return &services.OrderCreatedEvent{
		OrderID:        orderID,
		UserID:         "550e8400-e29b-41d4-a716-446655440000", // Realistic UUID from Keycloak
		EventID:        "taylor_swift_concert_2025",
		SessionID:      "evening_session_main",
		Status:         "CONFIRMED",
		SubTotal:       180.00,
		DiscountID:     "VIP_DISCOUNT",
		DiscountCode:   "VIP10",
		DiscountAmount: 18.00,
		Price:          162.00,
		CreatedAt:      timestamp.Format(time.RFC3339),
		PaymentAT:      timestamp.Add(2 * time.Minute).Format(time.RFC3339),
		Tickets: []services.Ticket{
			{
				TicketID:        "ticket_vip_001",
				OrderID:         orderID,
				SeatID:          "VIP_A1",
				SeatLabel:       "VIP Section A, Row 1, Seat 1",
				Colour:          "#FFD700",
				TierID:          "vip_tier",
				TierName:        "VIP Gold",
				PriceAtPurchase: 90.00,
				IssuedAt:        timestamp.Format(time.RFC3339),
				CheckedIn:       false,
			},
			{
				TicketID:        "ticket_vip_002",
				OrderID:         orderID,
				SeatID:          "VIP_A2",
				SeatLabel:       "VIP Section A, Row 1, Seat 2",
				Colour:          "#FFD700",
				TierID:          "vip_tier",
				TierName:        "VIP Gold",
				PriceAtPurchase: 90.00,
				IssuedAt:        timestamp.Format(time.RFC3339),
				CheckedIn:       false,
			},
		},
	}
}

// printOrderSummary displays order details
func printOrderSummary(order *services.OrderCreatedEvent) {
	fmt.Printf("   📋 Order: %s\n", order.OrderID)
	fmt.Printf("   👤 User UUID: %s\n", order.UserID)
	fmt.Printf("   🎫 Event: %s\n", order.EventID)
	fmt.Printf("   💰 Price: $%.2f (Original: $%.2f, Discount: $%.2f)\n",
		order.Price, order.SubTotal, order.DiscountAmount)
	fmt.Printf("   🎟️ Tickets: %d\n", len(order.Tickets))

	for i, ticket := range order.Tickets {
		fmt.Printf("      %d. %s - $%.2f\n", i+1, ticket.SeatLabel, ticket.PriceAtPurchase)
	}
}

// createOrGetSubscriber creates or retrieves a subscriber using UUID (simulating Keycloak lookup)
func createOrGetSubscriber(db *sql.DB, userID string) (*models.Subscriber, error) {
	// Try to get existing subscriber by user_id first
	query := `SELECT subscriber_id, user_id, subscriber_mail, created_at FROM subscribers WHERE user_id = $1`

	var subscriber models.Subscriber
	err := db.QueryRow(query, userID).Scan(
		&subscriber.SubscriberID,
		&subscriber.UserID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// Simulate Keycloak lookup to get email for the UUID
		// In real implementation, this would call Keycloak API
		email := "isurumuni.22@cse.mrt.ac.lk" // Simulated email from Keycloak

		fmt.Printf("   🔍 User %s not found in database, simulating Keycloak lookup...\n", userID)
		fmt.Printf("   📧 Keycloak returned email: %s\n", email)

		// Create new subscriber with both user_id and email
		// Handle case where email exists but user_id is different (update scenario)
		insertQuery := `
			INSERT INTO subscribers (user_id, subscriber_mail) 
			VALUES ($1, $2) 
			ON CONFLICT (subscriber_mail) DO UPDATE SET user_id = EXCLUDED.user_id
			RETURNING subscriber_id, user_id, subscriber_mail, created_at`

		err = db.QueryRow(insertQuery, userID, email).Scan(
			&subscriber.SubscriberID,
			&subscriber.UserID,
			&subscriber.SubscriberMail,
			&subscriber.CreatedAt,
		)
	} else if err == nil {
		fmt.Printf("   ✅ Found existing subscriber for UUID %s\n", userID)
	}

	return &subscriber, err
}

// addSubscription adds a subscription record
func addSubscription(db *sql.DB, subscriberID int, category string, targetID int) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, category, target_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (subscriber_id, category, target_id) DO NOTHING`

	_, err := db.Exec(query, subscriberID, category, targetID)
	return err
}

// verifySubscriber checks subscriber data in database
func verifySubscriber(db *sql.DB, subscriberID int) {
	// Check subscriber exists
	var email string
	var createdAt time.Time
	err := db.QueryRow(
		"SELECT subscriber_mail, created_at FROM subscribers WHERE subscriber_id = $1",
		subscriberID,
	).Scan(&email, &createdAt)

	if err != nil {
		fmt.Printf("   ❌ Subscriber verification failed: %v\n", err)
		return
	}

	fmt.Printf("   ✅ Subscriber verified: %s (created: %s)\n", email, createdAt.Format("2006-01-02 15:04:05"))

	// Check subscriptions
	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM subscriptions WHERE subscriber_id = $1",
		subscriberID,
	).Scan(&count)

	if err != nil {
		fmt.Printf("   ⚠️ Subscription count error: %v\n", err)
	} else {
		fmt.Printf("   📋 Active subscriptions: %d\n", count)
	}
}

func main() {
	TestOrderEmail()
}
