package services

import (
	"fmt"
	"log"
	"ms-scheduling/internal/models"
	"net/url"
	"strings"
	"time"
)

// buildSessionStartReminderEmail creates the email content for session start reminders (1 day before)
func (s *SubscriberService) buildSessionStartReminderEmail(subscriber models.Subscriber, sessionInfo *SessionReminderInfo) (string, string) {
	// Convert timestamps to readable format
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)
	endTime := models.MicroTimestampToTime(sessionInfo.EndTime)

	// Get subscriber name if possible
	subscriberName := s.getSubscriberName(subscriber)

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Your Event"
	}

	subject := fmt.Sprintf("🔔 Reminder: %s is tomorrow!", eventTitle)

	// Calculate session duration
	durationStr := s.formatDuration(startTime, endTime)

	// Format date and time more user-friendly
	dateStr := startTime.Format("Monday, January 2, 2006")
	startTimeStr := startTime.Format("3:04 PM")
	endTimeStr := endTime.Format("3:04 PM")

	// Generate calendar links
	calendarMsg := s.generateCalendarLinks(sessionInfo, eventTitle, startTime, endTime)

	// Build HTML email body
	var body strings.Builder
	body.WriteString(fmt.Sprintf("<h2>Hello %s!</h2>", subscriberName))
	body.WriteString(fmt.Sprintf("<p>This is a reminder that <strong>%s</strong> is happening tomorrow!</p>", eventTitle))
	body.WriteString("<p><strong>📅 Event Details:</strong></p>")
	body.WriteString("<ul>")
	body.WriteString(fmt.Sprintf("<li><strong>Date:</strong> %s</li>", dateStr))
	body.WriteString(fmt.Sprintf("<li><strong>Time:</strong> %s to %s</li>", startTimeStr, endTimeStr))
	body.WriteString(fmt.Sprintf("<li><strong>Duration:</strong> %s</li>", durationStr))
	if sessionInfo.VenueDetails != "" {
		body.WriteString(fmt.Sprintf("<li><strong>Venue:</strong> %s</li>", sessionInfo.VenueDetails))
	}
	body.WriteString("</ul>")
	body.WriteString("<p>We look forward to seeing you there!</p>")
	body.WriteString(calendarMsg)
	body.WriteString("<p><em>This is an automated reminder message. Please do not reply to this email.</em></p>")

	return subject, body.String()
}

// buildSessionSalesReminderEmail creates the email content for session sales start reminders
func (s *SubscriberService) buildSessionSalesReminderEmail(subscriber models.Subscriber, sessionInfo *SessionReminderInfo) (string, string) {
	// Convert timestamps to readable format
	salesStartTime := models.MicroTimestampToTime(sessionInfo.SalesStartTime)
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)

	// Get subscriber name if possible
	subscriberName := s.getSubscriberName(subscriber)

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Event"
	}

	subject := fmt.Sprintf("🎟️ Tickets for %s will be available soon!", eventTitle)

	// Format date and time more user-friendly
	salesDateStr := salesStartTime.Format("Monday, January 2, 2006")
	salesTimeStr := salesStartTime.Format("3:04 PM")
	eventDateStr := startTime.Format("Monday, January 2, 2006")

	// Build HTML email body
	var body strings.Builder
	body.WriteString(fmt.Sprintf("<h2>Hello %s!</h2>", subscriberName))
	body.WriteString(fmt.Sprintf("<p><strong>Tickets for %s will be available in 30 minutes!</strong></p>", eventTitle))
	body.WriteString("<p>Don't miss your chance to secure your spot.</p>")
	body.WriteString("<p><strong>🎫 Ticket Sales Information:</strong></p>")
	body.WriteString("<ul>")
	body.WriteString(fmt.Sprintf("<li><strong>Sales Start:</strong> %s at %s</li>", salesDateStr, salesTimeStr))
	body.WriteString(fmt.Sprintf("<li><strong>Event Date:</strong> %s</li>", eventDateStr))
	body.WriteString("</ul>")

	// Add purchase link if we have one
	body.WriteString("<p>")
	body.WriteString(fmt.Sprintf("<a href=\"https://ticketly.com/events/%s/sessions/%s\" style=\"background-color:#4CAF50;color:white;padding:10px 20px;text-align:center;text-decoration:none;display:inline-block;border-radius:5px;font-weight:bold;\">Buy Tickets</a>",
		sessionInfo.EventID, sessionInfo.SessionID))
	body.WriteString("</p>")

	body.WriteString("<p>Be ready to purchase as soon as tickets are available!</p>")
	body.WriteString("<p><em>This is an automated notification. Please do not reply to this email.</em></p>")

	return subject, body.String()
}

// Helper method to get subscriber name
func (s *SubscriberService) getSubscriberName(subscriber models.Subscriber) string {
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

	return subscriberName
}

// Helper method to format duration
func (s *SubscriberService) formatDuration(startTime, endTime time.Time) string {
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

	return durationStr
}

// Helper method to generate calendar links
func (s *SubscriberService) generateCalendarLinks(sessionInfo *SessionReminderInfo, eventTitle string, startTime, endTime time.Time) string {
	calendarMsg := "\n<p><strong>📱 Add to Calendar:</strong> "
	calendarMsg += fmt.Sprintf("<a href=\"https://calendar.google.com/calendar/render?action=TEMPLATE&text=%s&dates=%s/%s&details=%s at %s&location=%s\">Google Calendar</a> | ",
		url.QueryEscape(eventTitle),
		startTime.Format("20060102T150405"),
		endTime.Format("20060102T150405"),
		url.QueryEscape(eventTitle),
		url.QueryEscape(sessionInfo.VenueDetails),
		url.QueryEscape(sessionInfo.VenueDetails))
	calendarMsg += fmt.Sprintf("<a href=\"webcal://ticketly.com/calendar/event-%s.ics\">Apple Calendar</a></p>", sessionInfo.SessionID)

	return calendarMsg
}
