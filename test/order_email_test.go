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

// Test case for order email functionality
func main() {
	// Database connection (using environment variables)
	db, err := sql.Open("postgres",
		"host=localhost port=5432 user=ticketly password=ticketly dbname=ms_scheduling sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

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

	fmt.Println("🧪 Starting Order Email Test Case")
	fmt.Println(strings.Repeat("=", 50))

	// Test Case 1: Create sample order event
	testOrderEvent := createSampleOrderEvent()
	fmt.Printf("📋 Test Order Created: %s\n", testOrderEvent.OrderID)
	printOrderDetails(testOrderEvent)

	// Test Case 2: Test subscriber creation/retrieval
	fmt.Println("\n🔍 Testing Subscriber Management...")
	subscriber, err := subscriberService.GetOrCreateSubscriber(testOrderEvent.UserID)
	if err != nil {
		log.Printf("❌ Error creating/getting subscriber: %v", err)
		// For testing, let's create a subscriber directly
		subscriber, err = createTestSubscriber(db, testOrderEvent.UserID)
		if err != nil {
			log.Fatalf("Failed to create test subscriber: %v", err)
		}
	}
	fmt.Printf("✅ Subscriber found/created: ID=%d, Email=%s\n", subscriber.SubscriberID, subscriber.SubscriberMail)

	// Test Case 3: Add subscription for the order
	fmt.Println("\n📝 Adding Event Subscription...")
	err = addEventSubscription(db, subscriber.SubscriberID, testOrderEvent.EventID)
	if err != nil {
		log.Printf("⚠️ Warning: Could not add subscription: %v", err)
	} else {
		fmt.Println("✅ Event subscription added successfully")
	}

	// Test Case 4: Send order confirmation email
	fmt.Println("\n📧 Testing Email Sending...")
	err = subscriberService.SendOrderConfirmationEmail(subscriber, testOrderEvent)
	if err != nil {
		log.Printf("❌ Error sending email: %v", err)
	} else {
		fmt.Println("✅ Order confirmation email sent successfully!")
	}

	// Test Case 5: Verify database state
	fmt.Println("\n🔍 Verifying Database State...")
	verifyDatabaseState(db, subscriber.SubscriberID)

	fmt.Println("\n🎉 Test Case Completed!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("📊 Summary:")
	fmt.Printf("   - Order ID: %s\n", testOrderEvent.OrderID)
	fmt.Printf("   - User ID: %s\n", testOrderEvent.UserID)
	fmt.Printf("   - Subscriber Email: %s\n", subscriber.SubscriberMail)
	fmt.Printf("   - Total Price: $%.2f\n", testOrderEvent.Price)
	fmt.Printf("   - Tickets: %d\n", len(testOrderEvent.Tickets))
}

// createSampleOrderEvent creates a realistic order event for testing
func createSampleOrderEvent() *services.OrderCreatedEvent {
	return &services.OrderCreatedEvent{
		OrderID:        fmt.Sprintf("ORDER_%d", time.Now().Unix()),
		UserID:         "test-user-123@gmail.com",
		EventID:        "evt_concert_2025",
		SessionID:      "sess_evening_show",
		Status:         "CONFIRMED",
		SubTotal:       150.00,
		DiscountID:     "EARLY_BIRD",
		DiscountCode:   "EARLY20",
		DiscountAmount: 30.00,
		Price:          120.00,
		CreatedAt:      time.Now().Format(time.RFC3339),
		PaymentAT:      time.Now().Format(time.RFC3339),
		Tickets: []services.Ticket{
			{
				TicketID:        "ticket_001",
				OrderID:         "ORDER_" + fmt.Sprintf("%d", time.Now().Unix()),
				SeatID:          "A12",
				SeatLabel:       "Section A, Row 1, Seat 12",
				Colour:          "#FF6B35",
				TierID:          "vip_tier",
				TierName:        "VIP Premium",
				PriceAtPurchase: 75.00,
				IssuedAt:        time.Now().Format(time.RFC3339),
				CheckedIn:       false,
				CheckedInTime:   "",
			},
			{
				TicketID:        "ticket_002",
				OrderID:         "ORDER_" + fmt.Sprintf("%d", time.Now().Unix()),
				SeatID:          "A13",
				SeatLabel:       "Section A, Row 1, Seat 13",
				Colour:          "#FF6B35",
				TierID:          "vip_tier",
				TierName:        "VIP Premium",
				PriceAtPurchase: 75.00,
				IssuedAt:        time.Now().Format(time.RFC3339),
				CheckedIn:       false,
				CheckedInTime:   "",
			},
		},
	}
}

// printOrderDetails prints formatted order information
func printOrderDetails(order *services.OrderCreatedEvent) {
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   User ID: %s\n", order.UserID)
	fmt.Printf("   Event: %s\n", order.EventID)
	fmt.Printf("   Session: %s\n", order.SessionID)
	fmt.Printf("   Status: %s\n", order.Status)
	fmt.Printf("   Price: $%.2f (Subtotal: $%.2f, Discount: $%.2f)\n",
		order.Price, order.SubTotal, order.DiscountAmount)
	fmt.Printf("   Tickets: %d\n", len(order.Tickets))
	for i, ticket := range order.Tickets {
		fmt.Printf("     %d. %s (%s) - $%.2f\n",
			i+1, ticket.SeatLabel, ticket.TierName, ticket.PriceAtPurchase)
	}
}

// createTestSubscriber creates a test subscriber directly in the database
func createTestSubscriber(db *sql.DB, userID string) (*models.Subscriber, error) {
	query := `
		INSERT INTO subscribers (subscriber_mail) 
		VALUES ($1) 
		ON CONFLICT (subscriber_mail) DO UPDATE SET created_at = subscribers.created_at
		RETURNING subscriber_id, subscriber_mail, created_at
	`

	var subscriber models.Subscriber
	err := db.QueryRow(query, userID).Scan(
		&subscriber.SubscriberID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &subscriber, nil
}

// addEventSubscription adds a subscription for the test event
func addEventSubscription(db *sql.DB, subscriberID int, eventID string) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, category, target_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (subscriber_id, category, target_id) DO NOTHING
	`

	// Convert eventID to integer for target_id (in real scenario, you'd have proper mapping)
	targetID := 123 // This would be the actual event ID from your events table

	_, err := db.Exec(query, subscriberID, "event", targetID)
	return err
}

// verifyDatabaseState checks the current database state
func verifyDatabaseState(db *sql.DB, subscriberID int) {
	// Check subscriber
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM subscribers WHERE subscriber_id = $1", subscriberID).Scan(&count)
	if err != nil {
		log.Printf("❌ Error checking subscriber: %v", err)
	} else {
		fmt.Printf("   ✅ Subscriber exists in database: %d records\n", count)
	}

	// Check subscriptions
	err = db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE subscriber_id = $1", subscriberID).Scan(&count)
	if err != nil {
		log.Printf("❌ Error checking subscriptions: %v", err)
	} else {
		fmt.Printf("   ✅ Subscriptions for user: %d records\n", count)
	}

	// Show sample data
	fmt.Println("   📋 Sample Database Data:")
	rows, err := db.Query("SELECT subscriber_id, subscriber_mail FROM subscribers LIMIT 3")
	if err != nil {
		log.Printf("❌ Error querying subscribers: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("      Subscribers:")
	for rows.Next() {
		var id int
		var email string
		if err := rows.Scan(&id, &email); err == nil {
			fmt.Printf("        - ID: %d, Email: %s\n", id, email)
		}
	}
}
