package templates

import (
	"fmt"
	"time"

	"ms-scheduling/internal/email"
	"ms-scheduling/internal/email/builders"
	"ms-scheduling/internal/models"
)

// GenerateEventCreatedEmail generates an email for event creation/approval
func GenerateEventCreatedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#10B981")

	builder.SetHeader("🎊 New Event Published!", "An exciting new event is now available")

	builder.AddInfoBox(
		fmt.Sprintf("<strong>%s</strong> has been published and is now accepting registrations!", event.Title),
		"success",
	)

	builder.AddSection("📋 Event Details", fmt.Sprintf(`
		<p><strong>%s</strong></p>
		<p>%s</p>
	`, event.Title, event.Description))

	if event.Overview != "" {
		builder.AddParagraph(event.Overview)
	}

	created := time.Unix(event.CreatedAt/1000000, 0)
	details := map[string]string{
		"Event ID":     event.ID,
		"Organization": organizationName,
		"Status":       event.Status,
		"Published":    created.Format("Monday, January 2, 2006"),
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("Sessions for this event will be announced soon. You'll receive notifications when they become available.")
	// builder.AddButton("View Event Details", fmt.Sprintf("https://ticketly.com/events/%s", event.ID))

	return email.EmailTemplate{
		Type:    email.EmailEventCreated,
		Subject: fmt.Sprintf("🎊 New Event: %s", event.Title),
		HTML:    builder.Build(),
	}
}

// GenerateEventUpdatedEmail generates an email for event updates
func GenerateEventUpdatedEmail(before, after *models.Event, organizationName string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#4F46E5")

	builder.SetHeader("📝 Event Update", "An event you're following has been updated")

	// Detect what changed
	changes := detectEventChanges(before, after)
	if len(changes) > 0 {
		builder.AddSection("🔄 What Changed", buildChangesList(changes))
	}

	builder.AddSection("📋 Current Event Details", fmt.Sprintf(`
		<p><strong>%s</strong></p>
		<p>%s</p>
	`, after.Title, after.Description))

	updated := time.Unix(after.UpdatedAt/1000000, 0)
	details := map[string]string{
		"Event ID":     after.ID,
		"Organization": organizationName,
		"Status":       after.Status,
		"Last Updated": updated.Format("Monday, January 2, 2006 at 3:04 PM"),
	}
	builder.AddDetailsList(details)

	return email.EmailTemplate{
		Type:    email.EmailEventUpdated,
		Subject: fmt.Sprintf("Event Updated: %s", after.Title),
		HTML:    builder.Build(),
	}
}

// GenerateEventApprovedEmail generates an email when an event is approved
func GenerateEventApprovedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#10B981")

	builder.SetHeader("✅ Event Approved!", "Your event has been approved and published")

	builder.AddInfoBox(
		fmt.Sprintf("Congratulations! <strong>%s</strong> has been approved and is now visible to the public.", event.Title),
		"success",
	)

	builder.AddParagraph("Your event is now live and accepting registrations. You can start adding sessions and managing tickets.")

	details := map[string]string{
		"Event ID":     event.ID,
		"Event Title":  event.Title,
		"Organization": organizationName,
		"Status":       event.Status,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("Next steps:")
	builder.AddParagraph("• Add sessions to your event<br>• Set up ticket tiers and pricing<br>• Promote your event to reach more attendees")
	// builder.AddButton("Manage Event", fmt.Sprintf("https://ticketly.com/organizer/events/%s", event.ID))

	return email.EmailTemplate{
		Type:    email.EmailEventApproved,
		Subject: fmt.Sprintf("✅ Event Approved: %s", event.Title),
		HTML:    builder.Build(),
	}
}

// GenerateEventRejectedEmail generates an email when an event is rejected
func GenerateEventRejectedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#EF4444")

	builder.SetHeader("❌ Event Not Approved", "Your event submission requires attention")

	builder.AddInfoBox(
		fmt.Sprintf("Unfortunately, <strong>%s</strong> was not approved for publication.", event.Title),
		"error",
	)

	if event.RejectionReason != "" {
		builder.AddSection("📄 Reason for Rejection", fmt.Sprintf("<p>%s</p>", event.RejectionReason))
	}

	details := map[string]string{
		"Event ID":     event.ID,
		"Event Title":  event.Title,
		"Organization": organizationName,
		"Status":       event.Status,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("You can review the feedback, make necessary changes, and resubmit your event for approval.")
	// builder.AddButton("Edit Event", fmt.Sprintf("https://ticketly.com/organizer/events/%s/edit", event.ID))

	return email.EmailTemplate{
		Type:    email.EmailEventRejected,
		Subject: fmt.Sprintf("Event Submission Update: %s", event.Title),
		HTML:    builder.Build(),
	}
}

// GenerateEventCancelledEmail generates an email when an event is cancelled
func GenerateEventCancelledEmail(event *models.Event, organizationName string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#EF4444")

	builder.SetHeader("❌ Event Cancelled", "Important: An event has been cancelled")

	builder.AddInfoBox(
		"<strong>⚠️ This event has been cancelled and removed from the schedule.</strong>",
		"error",
	)

	builder.AddParagraph(fmt.Sprintf("We regret to inform you that <strong>%s</strong> has been cancelled.", event.Title))

	created := time.Unix(event.CreatedAt/1000000, 0)
	details := map[string]string{
		"Event ID":     event.ID,
		"Event Title":  event.Title,
		"Organization": organizationName,
		"Created On":   created.Format("Monday, January 2, 2006"),
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("<strong>Refund Information:</strong>")
	builder.AddParagraph("If you have purchased tickets for this event, you will be automatically refunded within 5-7 business days. You will receive a separate confirmation email once the refund is processed.")
	builder.AddParagraph("For any questions or concerns, please contact our support team.")

	return email.EmailTemplate{
		Type:    email.EmailEventCancelled,
		Subject: fmt.Sprintf("⚠️ Event Cancelled: %s", event.Title),
		HTML:    builder.Build(),
	}
}

// Helper functions

func detectEventChanges(before, after *models.Event) map[string]string {
	changes := make(map[string]string)

	if before.Title != after.Title {
		changes["Title"] = fmt.Sprintf("%s → %s", before.Title, after.Title)
	}

	if before.Description != after.Description {
		changes["Description"] = "Event description has been updated"
	}

	if before.Overview != after.Overview {
		changes["Overview"] = "Event overview has been updated"
	}

	if before.Status != after.Status {
		changes["Status"] = fmt.Sprintf("%s → %s", before.Status, after.Status)
	}

	if before.CategoryID != after.CategoryID {
		changes["Category"] = "Event category has been changed"
	}

	return changes
}
