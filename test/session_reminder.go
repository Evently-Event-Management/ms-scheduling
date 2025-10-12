package main

import (
	"fmt"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"time"
)

func main() {
	log.Println("🚀 Testing session reminder email functionality...")

	// Load configuration
	cfg := config.Load()

	// Create email service
	emailService := services.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.FromEmail, cfg.FromName)

	// Create test session reminder info
	// Using session ID that would have subscribers and event details
	sessionInfo := &services.SessionReminderInfo{
		SessionID:      "1000",
		EventID:        "456",
		EventTitle:     "An Example Event - Session Reminder",
		StartTime:      models.TimeToMicroTimestamp(time.Now().AddDate(0, 0, 1)), // Tomorrow
		EndTime:        models.TimeToMicroTimestamp(time.Now().AddDate(0, 0, 1).Add(2 * time.Hour)), // Tomorrow + 2 hours
		Status:         "ON_SALE",
		VenueDetails:   `{"name": "Main Auditorium", "address": "123 Event St, City"}`,
		SessionType:    "PHYSICAL",
		SalesStartTime: models.TimeToMicroTimestamp(time.Now().Add(-24 * time.Hour)), // Started yesterday
	}

	log.Printf("Testing session reminder for session %s (Event: %s)", sessionInfo.SessionID, sessionInfo.EventTitle)
	log.Printf("Session starts at: %s", models.MicroTimestampToTime(sessionInfo.StartTime).Format("2006-01-02 15:04:05"))

	// Create test subscribers
	userID1 := "test-user-123"
	userID2 := "test-user-456"
	
	testSubscribers := []models.Subscriber{
		{
			SubscriberID:   1,
			UserID:         &userID1,
			SubscriberMail: "isurumuni.22@cse.mrt.ac.lk", // User's email from conversation
			CreatedAt:      time.Now(),
		},
		{
			SubscriberID:   2,
			UserID:         &userID2,
			SubscriberMail: "user2@example.com",
			CreatedAt:      time.Now(),
		},
	}

	log.Printf("📧 Sending session reminder emails to %d test subscribers", len(testSubscribers))

	// Create a mock SubscriberService for testing
	subscriberService := &MockSubscriberService{emailService: emailService}
	
	// Send reminder emails
	err := subscriberService.SendSessionReminderEmails(testSubscribers, sessionInfo)
	if err != nil {
		log.Printf("❌ Error sending session reminder emails: %v", err)
		return
	}

	log.Println("✅ Session reminder notification emails sent successfully!")
	log.Println("📧 Check your inbox at isurumuni.22@cse.mrt.ac.lk for the session reminder!")
	log.Printf("📅 Reminder: Session starts tomorrow at %s", models.MicroTimestampToTime(sessionInfo.StartTime).Format("2006-01-02 15:04:05"))
}

// MockSubscriberService implements just the methods we need for testing
type MockSubscriberService struct {
	emailService *services.EmailService
}

// SendSessionReminderEmails sends reminder emails to all subscribers
func (m *MockSubscriberService) SendSessionReminderEmails(subscribers []models.Subscriber, sessionInfo *services.SessionReminderInfo) error {
	log.Printf("Sending session reminder emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := m.buildSessionReminderEmail(subscriber, sessionInfo)

		err := m.emailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending session reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// buildSessionReminderEmail creates the email content for session reminders
func (m *MockSubscriberService) buildSessionReminderEmail(subscriber models.Subscriber, sessionInfo *services.SessionReminderInfo) (string, string) {
	// Convert timestamps to readable format
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)
	endTime := models.MicroTimestampToTime(sessionInfo.EndTime)
	
	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Your Event"
	}

	subject := fmt.Sprintf("🔔 Reminder: %s starts tomorrow!", eventTitle)

	var body string
	body += fmt.Sprintf("Hello %s,\n\n", subscriber.SubscriberMail)
	body += "🔔 This is a friendly reminder that you have a session starting tomorrow!\n\n"
	
	body += "Session Details:\n"
	if sessionInfo.EventTitle != "" {
		body += fmt.Sprintf("• Event: %s\n", sessionInfo.EventTitle)
	}
	body += fmt.Sprintf("• Session Type: %s\n", sessionInfo.SessionType)
	body += fmt.Sprintf("• Status: %s\n", sessionInfo.Status)
	body += fmt.Sprintf("• Start Time: %s\n", startTime.Format("2006-01-02 15:04:05"))
	body += fmt.Sprintf("• End Time: %s\n", endTime.Format("2006-01-02 15:04:05"))
	body += fmt.Sprintf("• Session ID: %s\n", sessionInfo.SessionID)
	
	// Add venue details if available
	if sessionInfo.VenueDetails != "" {
		body += fmt.Sprintf("• Venue Details: %s\n", sessionInfo.VenueDetails)
	}

	body += fmt.Sprintf("\n⏰ The session starts in approximately 24 hours!\n")
	
	if sessionInfo.Status == "ON_SALE" {
		body += "\n🎫 Don't forget - this session is currently on sale. Make sure you have your tickets ready!\n"
	} else if sessionInfo.Status == "PENDING" {
		body += "\n⏳ This session is still pending. We'll update you if there are any changes.\n"
	}

	body += "\n📅 We recommend:"
	body += "\n• Set a reminder on your phone"
	body += "\n• Check the venue location and directions"
	body += "\n• Prepare any required documents or tickets"
	body += "\n• Plan your travel time with some buffer"

	body += "\n\nSee you tomorrow! 🎉"
	body += "\n\nBest regards,\nTicketly Team"

	return subject, body
}