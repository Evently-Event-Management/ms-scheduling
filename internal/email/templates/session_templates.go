package templates

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ms-scheduling/internal/email"
	"ms-scheduling/internal/email/builders"
	"ms-scheduling/internal/models"
)

// GenerateSessionCreatedEmail generates an email for session creation
func GenerateSessionCreatedEmail(session *models.EventSession, eventTitle string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#4F46E5")

	start := time.Unix(session.StartTime/1000000, 0)
	end := time.Unix(session.EndTime/1000000, 0)

	builder.SetHeader("🎉 New Session Available!", "A new session has been added to an event you're following")

	builder.AddInfoBox(
		fmt.Sprintf("<strong>Event:</strong> %s", eventTitle),
		"info",
	)

	details := map[string]string{
		"Session ID":   session.ID,
		"Event ID":     session.EventID,
		"Session Type": session.SessionType,
		"Status":       session.Status,
		"Start Time":   start.Format("Monday, January 2, 2006 at 3:04 PM"),
		"End Time":     end.Format("Monday, January 2, 2006 at 3:04 PM"),
		"Duration":     formatDuration(end.Sub(start)),
	}
	builder.AddDetailsList(details)

	if session.VenueDetails != "" {
		venueHTML := formatVenueDetails(session.VenueDetails)
		builder.AddSection("📍 Venue Information", venueHTML)
	}

	builder.AddDivider()
	builder.AddParagraph("Don't miss out! This session is now available for registration.")
	// builder.AddButton("View Session Details", fmt.Sprintf("https://ticketly.com/sessions/%s", session.ID))

	return email.EmailTemplate{
		Type:    email.EmailSessionCreated,
		Subject: fmt.Sprintf("New Session Available - %s", eventTitle),
		HTML:    builder.Build(),
	}
}

// GenerateSessionUpdatedEmail generates an email for session updates
func GenerateSessionUpdatedEmail(before, after *models.EventSession, eventTitle string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#4F46E5")

	builder.SetHeader("📝 Session Update", "A session you're following has been updated")

	builder.AddInfoBox(
		fmt.Sprintf("<strong>Event:</strong> %s", eventTitle),
		"info",
	)

	// Detect what changed
	changes := detectSessionChanges(before, after)
	if len(changes) > 0 {
		builder.AddSection("🔄 What Changed", buildChangesList(changes))
	}

	// Current details
	start := time.Unix(after.StartTime/1000000, 0)
	end := time.Unix(after.EndTime/1000000, 0)
	details := map[string]string{
		"Session ID":   after.ID,
		"Status":       after.Status,
		"Session Type": after.SessionType,
		"Start Time":   start.Format("Monday, January 2, 2006 at 3:04 PM"),
		"End Time":     end.Format("Monday, January 2, 2006 at 3:04 PM"),
	}
	builder.AddDetailsList(details)

	if after.VenueDetails != "" {
		venueHTML := formatVenueDetails(after.VenueDetails)
		builder.AddSection("📍 Venue Information", venueHTML)
	}

	return email.EmailTemplate{
		Type:    email.EmailSessionUpdated,
		Subject: fmt.Sprintf("Session Updated - %s", eventTitle),
		HTML:    builder.Build(),
	}
}

// GenerateSessionCancelledEmail generates an email for session cancellation
func GenerateSessionCancelledEmail(session *models.EventSession, eventTitle string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#EF4444")

	builder.SetHeader("❌ Session Cancelled", "Important: A session has been cancelled")

	builder.AddInfoBox(
		"<strong>⚠️ This session has been cancelled or removed from the schedule.</strong>",
		"error",
	)

	builder.AddParagraph(fmt.Sprintf("The session for <strong>%s</strong> has been cancelled.", eventTitle))

	start := time.Unix(session.StartTime/1000000, 0)
	details := map[string]string{
		"Session ID":      session.ID,
		"Event ID":        session.EventID,
		"Scheduled Time":  start.Format("Monday, January 2, 2006 at 3:04 PM"),
		"Previous Status": session.Status,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("If you have purchased tickets for this session, you will receive a separate email regarding refunds.")
	builder.AddParagraph("For any questions or concerns, please contact our support team.")

	return email.EmailTemplate{
		Type:    email.EmailSessionCancelled,
		Subject: fmt.Sprintf("⚠️ Session Cancelled - %s", eventTitle),
		HTML:    builder.Build(),
	}
}

// GenerateSessionReminderEmail generates a reminder email before session starts
func GenerateSessionReminderEmail(session *models.EventSession, eventTitle string, hoursUntil int) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#F59E0B")

	reminderText := fmt.Sprintf("Reminder: Your session starts in %d hours!", hoursUntil)
	if hoursUntil == 24 {
		reminderText = "Reminder: Your session starts tomorrow!"
	} else if hoursUntil == 1 {
		reminderText = "Reminder: Your session starts in 1 hour!"
	}

	builder.SetHeader("⏰ Session Reminder", reminderText)

	start := time.Unix(session.StartTime/1000000, 0)
	end := time.Unix(session.EndTime/1000000, 0)

	builder.AddInfoBox(
		fmt.Sprintf("<strong>%s</strong><br>%s", eventTitle, start.Format("Monday, January 2, 2006 at 3:04 PM")),
		"warning",
	)

	details := map[string]string{
		"Session ID":   session.ID,
		"Session Type": session.SessionType,
		"Start Time":   start.Format("3:04 PM"),
		"End Time":     end.Format("3:04 PM"),
		"Duration":     formatDuration(end.Sub(start)),
	}
	builder.AddDetailsList(details)

	if session.VenueDetails != "" {
		venueHTML := formatVenueDetails(session.VenueDetails)
		builder.AddSection("📍 How to Get There", venueHTML)
	}

	builder.AddDivider()
	builder.AddParagraph("Please make sure to arrive early and have your tickets ready.")
	// builder.AddButton("View My Tickets", "https://ticketly.com/my-tickets")

	return email.EmailTemplate{
		Type:    email.EmailSessionReminder,
		Subject: fmt.Sprintf("⏰ Reminder: %s - Starting Soon!", eventTitle),
		HTML:    builder.Build(),
	}
}

// Helper functions

func formatVenueDetails(venueJSON string) string {
	var venue map[string]interface{}
	if err := json.Unmarshal([]byte(venueJSON), &venue); err != nil {
		return fmt.Sprintf("<p>%s</p>", venueJSON)
	}

	var parts []string
	for key, value := range venue {
		parts = append(parts, fmt.Sprintf("<strong>%s:</strong> %v", strings.Title(key), value))
	}
	return "<p>" + strings.Join(parts, "<br>") + "</p>"
}

func formatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours == 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, minutes)
}

func detectSessionChanges(before, after *models.EventSession) map[string]string {
	changes := make(map[string]string)

	if before.StartTime != after.StartTime {
		beforeTime := time.Unix(before.StartTime/1000000, 0)
		afterTime := time.Unix(after.StartTime/1000000, 0)
		changes["Start Time"] = fmt.Sprintf("%s → %s",
			beforeTime.Format("Jan 2, 3:04 PM"),
			afterTime.Format("Jan 2, 3:04 PM"))
	}

	if before.EndTime != after.EndTime {
		beforeTime := time.Unix(before.EndTime/1000000, 0)
		afterTime := time.Unix(after.EndTime/1000000, 0)
		changes["End Time"] = fmt.Sprintf("%s → %s",
			beforeTime.Format("Jan 2, 3:04 PM"),
			afterTime.Format("Jan 2, 3:04 PM"))
	}

	if before.Status != after.Status {
		changes["Status"] = fmt.Sprintf("%s → %s", before.Status, after.Status)
	}

	if before.SessionType != after.SessionType {
		changes["Session Type"] = fmt.Sprintf("%s → %s", before.SessionType, after.SessionType)
	}

	if before.VenueDetails != after.VenueDetails {
		changes["Venue"] = "Venue details have been updated"
	}

	return changes
}

func buildChangesList(changes map[string]string) string {
	var items []string
	for key, value := range changes {
		items = append(items, fmt.Sprintf("<li><strong>%s:</strong> %s</li>", key, value))
	}
	return "<ul style='margin: 10px 0; padding-left: 20px;'>" + strings.Join(items, "") + "</ul>"
}
