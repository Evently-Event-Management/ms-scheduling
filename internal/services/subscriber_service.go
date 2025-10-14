package services

import (
	"fmt"
	"log"
	"ms-scheduling/internal/models"
	"net/url"
	"strings"
)

// SendSessionReminderEmails sends generic reminder emails to all subscribers
func (s *SubscriberService) SendSessionReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending generic session reminder emails to %d subscribers", len(subscribers))

	// Generate email template using our new template system
	emailTemplate := generateSessionStartReminderEmail(s.Config, sessionInfo)

	for _, subscriber := range subscribers {
		err := s.EmailService.SendEmail(subscriber.SubscriberMail, emailTemplate.Subject, emailTemplate.HTML)
		if err != nil {
			log.Printf("Error sending session reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// SendSessionStartReminderEmails sends session start reminder emails (1 day before)
func (s *SubscriberService) SendSessionStartReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending session START reminder emails to %d subscribers (1 day before)", len(subscribers))

	// Generate email template using our new template system
	emailTemplate := generateSessionStartReminderEmail(s.Config, sessionInfo)

	for _, subscriber := range subscribers {
		err := s.EmailService.SendEmail(subscriber.SubscriberMail, emailTemplate.Subject, emailTemplate.HTML)
		if err != nil {
			log.Printf("Error sending session start reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session start reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// SendSessionSalesReminderEmails sends sales start reminder emails (30 min before)
func (s *SubscriberService) SendSessionSalesReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending session SALES reminder emails to %d subscribers", len(subscribers))

	// Generate email template using our new template system
	emailTemplate := generateSessionSalesReminderEmail(s.Config, sessionInfo)

	for _, subscriber := range subscribers {
		err := s.EmailService.SendEmail(subscriber.SubscriberMail, emailTemplate.Subject, emailTemplate.HTML)
		if err != nil {
			log.Printf("Error sending sales start reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Sales start reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// Note: This function is being deprecated in favor of email templates in email_common_templates.go
// TODO: Update SendSessionReminderEmails to use GenerateEmailTemplate instead
// buildSessionReminderEmail creates the email content for session reminders
func (s *SubscriberService) buildSessionReminderEmail(subscriber models.Subscriber, sessionInfo *SessionReminderInfo) (string, string) {
	// Convert timestamps to readable format
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)
	endTime := models.MicroTimestampToTime(sessionInfo.EndTime)

	// Get subscriber name if possible
	subscriberName := ""
	if subscriber.UserID != nil && *subscriber.UserID != "" {
		// Try to get user details from Keycloak
		userDetails, err := s.KeycloakClient.GetUserDetails(*subscriber.UserID)
		if err == nil && userDetails != nil {
			if userDetails.FirstName != "" && userDetails.LastName != "" {
				subscriberName = fmt.Sprintf("%s %s", userDetails.FirstName, userDetails.LastName)
			} else if userDetails.FirstName != "" {
				subscriberName = userDetails.FirstName
			}
		} else {
			log.Printf("Failed to get Keycloak user details: %v", err)
		}
	}

	// Use email as fallback if name not available
	if subscriberName == "" {
		// Extract name from email if possible
		emailParts := strings.Split(subscriber.SubscriberMail, "@")
		subscriberName = emailParts[0]
	}

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Your Event"
	}

	subject := fmt.Sprintf("🔔 Reminder: %s is tomorrow!", eventTitle)

	// Calculate session duration
	duration := endTime.Sub(startTime)
	durationHours := int(duration.Hours())
	durationMinutes := int(duration.Minutes()) % 60

	// Format duration string
	var durationStr string
	if durationHours > 0 {
		if durationMinutes > 0 {
			durationStr = fmt.Sprintf("%d hours %d minutes", durationHours, durationMinutes)
		} else {
			durationStr = fmt.Sprintf("%d hours", durationHours)
		}
	} else {
		durationStr = fmt.Sprintf("%d minutes", durationMinutes)
	}

	// Format date and time more user-friendly
	dateStr := startTime.Format("Monday, January 2, 2006")
	startTimeStr := startTime.Format("3:04 PM")
	endTimeStr := endTime.Format("3:04 PM")

	// Generate calendar links
	calendarMsg := "\n<p><strong>📱 Add to Calendar:</strong> "
	calendarMsg += fmt.Sprintf("<a href=\"https://calendar.google.com/calendar/render?action=TEMPLATE&text=%s&dates=%s/%s&details=%s at %s&location=%s\">Google Calendar</a> | ",
		url.QueryEscape(eventTitle),
		startTime.Format("20060102T150405"),
		endTime.Format("20060102T150405"),
		url.QueryEscape(eventTitle),
		url.QueryEscape(sessionInfo.VenueDetails),
		url.QueryEscape(sessionInfo.VenueDetails))
	calendarMsg += fmt.Sprintf("<a href=\"webcal://ticketly.com/calendar/event-%s.ics\">Apple Calendar</a></p>", sessionInfo.SessionID)

	// Build HTML email body
	var body strings.Builder
	body.WriteString(fmt.Sprintf("<h2>Hello %s!</h2>", subscriberName))
	body.WriteString("<p><strong>🔔 This is a friendly reminder that you have a session starting tomorrow!</strong></p>")

	body.WriteString("<div style=\"background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0;\">")
	body.WriteString("<h3 style=\"color: #007bff; margin-top: 0;\">Session Details</h3>")

	// Event info section
	body.WriteString("<div style=\"margin-bottom: 20px;\">")
	if sessionInfo.EventTitle != "" {
		body.WriteString(fmt.Sprintf("<h4 style=\"margin-bottom: 5px;\">%s</h4>", sessionInfo.EventTitle))
	}
	body.WriteString(fmt.Sprintf("<p><strong>Type:</strong> %s</p>", sessionInfo.SessionType))
	body.WriteString(fmt.Sprintf("<p><strong>Date:</strong> %s</p>", dateStr))
	body.WriteString(fmt.Sprintf("<p><strong>Time:</strong> %s - %s (%s)</p>", startTimeStr, endTimeStr, durationStr))

	// Add venue details if available
	if sessionInfo.VenueDetails != "" {
		body.WriteString(fmt.Sprintf("<p><strong>Location:</strong> %s</p>", sessionInfo.VenueDetails))
	}

	// Status-specific messaging
	if sessionInfo.Status == "ON_SALE" {
		body.WriteString("<p><span style=\"color: #28a745; font-weight: bold;\">🎫 TICKETS ON SALE NOW</span> - Don't forget to purchase your tickets!</p>")
	} else if sessionInfo.Status == "SOLD_OUT" {
		body.WriteString("<p><span style=\"color: #dc3545; font-weight: bold;\">SOLD OUT</span> - This session is sold out.</p>")
	} else if sessionInfo.Status == "PENDING" {
		body.WriteString("<p><span style=\"color: #ffc107; font-weight: bold;\">⏳ PENDING CONFIRMATION</span> - We'll update you if there are any changes.</p>")
	} else if sessionInfo.Status == "CONFIRMED" {
		body.WriteString("<p><span style=\"color: #28a745; font-weight: bold;\">✅ CONFIRMED</span> - This session is confirmed to take place as scheduled.</p>")
	}
	body.WriteString("</div>")

	// Session ID for reference
	body.WriteString(fmt.Sprintf("<p style=\"font-size: 12px; color: #6c757d;\">Reference #: %s</p>", sessionInfo.SessionID))
	body.WriteString("</div>")

	// Add countdown and calendar links
	body.WriteString("<p style=\"font-size: 18px; font-weight: bold; color: #007bff;\">⏰ This session starts in approximately 24 hours!</p>")
	body.WriteString(calendarMsg)

	// Add checklist and recommendations
	body.WriteString("<div style=\"background-color: #e9ecef; padding: 15px; border-radius: 5px; margin: 20px 0;\">")
	body.WriteString("<h4>📋 Pre-Session Checklist:</h4>")
	body.WriteString("<ul>")
	body.WriteString("<li>Set a reminder on your phone</li>")
	body.WriteString("<li>Check the venue location and plan your route</li>")
	body.WriteString("<li>Prepare any required documents or tickets</li>")
	body.WriteString("<li>Plan your travel time with extra buffer</li>")
	body.WriteString("</ul>")
	body.WriteString("</div>")

	body.WriteString("<p>We're excited to see you tomorrow! 🎉</p>")
	body.WriteString("<p>Best regards,<br>The Ticketly Team</p>")

	// Unsubscribe option
	body.WriteString("<p style=\"font-size: 12px; color: #6c757d; margin-top: 30px;\">")
	body.WriteString(fmt.Sprintf("To unsubscribe from these notifications, <a href=\"https://ticketly.com/unsubscribe/%s\">click here</a>.", sessionInfo.SessionID))
	body.WriteString("</p>")

	return subject, body.String()
}

// SessionReminderInfo holds session information for reminder emails
type SessionReminderInfo struct {
	SessionID      string
	EventID        string
	EventTitle     string
	StartTime      int64
	EndTime        int64
	Status         string
	VenueDetails   string
	SessionType    string
	SalesStartTime int64
}
