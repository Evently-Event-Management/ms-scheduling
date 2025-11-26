package templates

import (
	"encoding/json"
	"fmt"
	"time"

	"ms-scheduling/internal/email"
	"ms-scheduling/internal/email/builders"
	"ms-scheduling/internal/models"
)

// Helper to generate venue HTML with map
func generateEventVenueHTML(venueJSON string) string {
	if venueJSON == "" {
		return ""
	}

	type VenueDetails struct {
		Name       string `json:"name"`
		Address    string `json:"address"`
		OnlineLink string `json:"onlineLink"`
		Location   struct {
			X           float64   `json:"x"`
			Y           float64   `json:"y"`
			Coordinates []float64 `json:"coordinates"`
			Type        string    `json:"type"`
		} `json:"location"`
	}

	var venue VenueDetails
	if err := json.Unmarshal([]byte(venueJSON), &venue); err != nil {
		return fmt.Sprintf(`<p><strong>📍 Venue:</strong> %s</p>`, venueJSON)
	}

	// Check if it's an online event
	if venue.OnlineLink != "" {
		return fmt.Sprintf(`
			<div style="margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-radius: 8px;">
				<h3 style="color: #2c3e50; margin-top: 0;">💻 Online Event</h3>
				<p style="margin: 0 0 10px 0;"><strong>%s</strong></p>
				<p style="text-align: center; margin-top: 10px;">
					<a href="%s" style="display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">
						Join Online Event
					</a>
				</p>
			</div>
		`, venue.Name, venue.OnlineLink)
	}

	// Physical event with location
	lat := venue.Location.Y
	lng := venue.Location.X

	if lat != 0 && lng != 0 {
		mapURL := fmt.Sprintf("https://maps.google.com/maps?q=%f,%f&z=15&output=embed", lat, lng)
		directionsURL := fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%f,%f", lat, lng)

		addressHTML := ""
		if venue.Address != "" {
			addressHTML = fmt.Sprintf("<p style=\"margin: 0 0 10px 0; color: #666;\">📮 %s</p>", venue.Address)
		}

		return fmt.Sprintf(`
			<div style="margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-radius: 8px;">
				<h3 style="color: #2c3e50; margin-top: 0;">📍 Venue Location</h3>
				<p style="margin: 0 0 10px 0;"><strong>%s</strong></p>
				%s
				<div style="margin: 15px 0;">
					<iframe 
						width="100%%" 
						height="250" 
						frameborder="0" 
						style="border:0; border-radius: 8px;" 
						src="%s"
						allowfullscreen>
					</iframe>
				</div>
				<p style="text-align: center; margin-top: 10px;">
					<a href="%s" style="display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">
						🗺️ Get Directions
					</a>
				</p>
			</div>
		`, venue.Name, addressHTML, mapURL, directionsURL)
	}

	// No coordinates, just show text
	addressHTML := ""
	if venue.Address != "" {
		addressHTML = fmt.Sprintf("<p>📮 %s</p>", venue.Address)
	}

	return fmt.Sprintf(`
		<div style="margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-radius: 8px;">
			<h3 style="color: #2c3e50; margin-top: 0;">📍 Venue</h3>
			<p style="margin: 0 0 10px 0;"><strong>%s</strong></p>
			%s
		</div>
	`, venue.Name, addressHTML)
}

// GenerateEventCreatedEmail generates an email for event creation/approval
func GenerateEventCreatedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	// Generate organization info
	var orgHTML string
	if organizationName != "" {
		orgHTML = fmt.Sprintf(`
			<div style="margin: 15px 0; padding: 10px; background-color: #f8f9fa; border-radius: 8px;">
				<p style="margin: 0; font-size: 12px; color: #666;">Organized by</p>
				<p style="margin: 0; font-weight: bold;">%s</p>
			</div>
		`, organizationName)
	}

	created := time.Unix(event.CreatedAt/1000000, 0)
	createdStr := created.Format("Monday, January 2, 2006")

	content := fmt.Sprintf(`
		<div class="header">
			<h1>🎊 New Event Published!</h1>
		</div>
		<div class="content">
			<div class="alert alert-success" style="padding: 15px; background-color: #d4edda; border-left: 4px solid #10B981; border-radius: 4px; margin: 20px 0; color: #155724;">
				<strong style="font-size: 18px;">%s is now live and accepting registrations!</strong>
			</div>
			%s
			<p>Hello,</p>
			<p>An exciting new event has been published and is now available for registration.</p>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-left: 4px solid #10B981; border-radius: 4px;">
				<h3 style="margin-top: 0; color: #2c3e50;">📋 Event Details</h3>
				<p style="margin: 0; line-height: 1.6;"><strong>%s</strong></p>
				<p style="margin: 10px 0 0 0; line-height: 1.6; color: #666;">%s</p>
			</div>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #fff; border: 1px solid #dee2e6; border-radius: 8px;">
				<h3 style="color: #2c3e50;">ℹ️ Event Information</h3>
				<ul style="list-style: none; padding: 0;">
					<li style="margin: 10px 0;"><strong>📌 Event ID:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>🏢 Organization:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>✅ Status:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>📅 Published:</strong> %s</li>
				</ul>
			</div>
			
			<p style="text-align: center; margin: 30px 0;">
				<a href="https://ticketly.dpiyumal.me/events/%s" style="display: inline-block; padding: 12px 30px; background-color: #10B981; color: white; text-decoration: none; border-radius: 5px; font-weight: bold; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
					View Event Details
				</a>
			</p>
			
			<p>Sessions for this event will be announced soon. You'll receive notifications when they become available.</p>
		</div>
	`, event.Title, orgHTML, event.Title, event.Description, event.ID, organizationName, event.Status, createdStr, event.ID)

	html := wrapEventEmailHTML(event.Title, "🎊 New Event Published", content)

	return email.EmailTemplate{
		Type:    email.EmailEventCreated,
		Subject: fmt.Sprintf("🎊 New Event: %s", event.Title),
		HTML:    html,
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
	// Note: event parameter only contains basic DB fields from Debezium CDC
	// For full event details (cover photos, venue), would need to call event-query service
	// For now, we'll create a clean, professional email with available data
	
	// Generate organization info
	var orgHTML string
	if organizationName != "" {
		orgHTML = fmt.Sprintf(`
			<div style="margin: 15px 0; padding: 10px; background-color: #f8f9fa; border-radius: 8px;">
				<p style="margin: 0; font-size: 12px; color: #666;">Organized by</p>
				<p style="margin: 0; font-weight: bold;">%s</p>
			</div>
		`, organizationName)
	}

	created := time.Unix(event.CreatedAt/1000000, 0)
	createdStr := created.Format("Monday, January 2, 2006")

	content := fmt.Sprintf(`
		<div class="header">
			<h1>✅ Event Approved!</h1>
		</div>
		<div class="content">
			<div class="alert alert-success" style="padding: 15px; background-color: #d4edda; border-left: 4px solid #28a745; border-radius: 4px; margin: 20px 0; color: #155724;">
				<strong style="font-size: 18px;">🎉 Congratulations! Your event has been approved and is now live!</strong>
			</div>
			%s
			<p>Hello,</p>
			<p>Great news! <strong>%s</strong> has been reviewed and approved. Your event is now visible to the public and accepting registrations.</p>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-left: 4px solid #28a745; border-radius: 4px;">
				<h3 style="margin-top: 0; color: #2c3e50;">About Your Event</h3>
				<p style="margin: 0; line-height: 1.6;"><strong>%s</strong></p>
				<p style="margin: 10px 0 0 0; line-height: 1.6; color: #666;">%s</p>
			</div>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #fff; border: 1px solid #dee2e6; border-radius: 8px;">
				<h3 style="color: #2c3e50;">📋 Event Information</h3>
				<ul style="list-style: none; padding: 0;">
					<li style="margin: 10px 0;"><strong>📌 Event ID:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>🏢 Organization:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>✅ Status:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>📅 Published:</strong> %s</li>
				</ul>
			</div>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #e7f3ff; border-left: 4px solid #007bff; border-radius: 4px;">
				<h3 style="margin-top: 0; color: #004085;">🚀 Next Steps</h3>
				<ul style="color: #004085; line-height: 1.8;">
					<li>✓ Add sessions and schedule to your event</li>
					<li>✓ Set up ticket tiers and pricing</li>
					<li>✓ Configure payment and refund policies</li>
					<li>✓ Promote your event to reach more attendees</li>
					<li>✓ Monitor registrations and ticket sales</li>
				</ul>
			</div>
			
			<p style="text-align: center; margin: 30px 0;">
				<a href="https://ticketly.dpiyumal.me/organizer/events/%s" style="display: inline-block; padding: 12px 30px; background-color: #28a745; color: white; text-decoration: none; border-radius: 5px; font-weight: bold; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
					Manage Your Event
				</a>
			</p>
			
			<p style="text-align: center; margin: 30px 0; font-size: 16px;">Your event is now live and ready for registrations! 🎊</p>
		</div>
	`, orgHTML, event.Title, event.Title, event.Description, event.ID, organizationName, event.Status, createdStr, event.ID)

	// Wrap in HTML document with inline styles
	html := wrapEventEmailHTML(event.Title, "✅ Event Approved", content)

	return email.EmailTemplate{
		Type:    email.EmailEventApproved,
		Subject: fmt.Sprintf("✅ Event Approved: %s", event.Title),
		HTML:    html,
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
	// Generate organization info
	var orgHTML string
	if organizationName != "" {
		orgHTML = fmt.Sprintf(`
			<div style="margin: 15px 0; padding: 10px; background-color: #f8f9fa; border-radius: 8px;">
				<p style="margin: 0; font-size: 12px; color: #666;">Organized by</p>
				<p style="margin: 0; font-weight: bold;">%s</p>
			</div>
		`, organizationName)
	}

	created := time.Unix(event.CreatedAt/1000000, 0)
	createdStr := created.Format("Monday, January 2, 2006")

	content := fmt.Sprintf(`
		<div class="header">
			<h1>❌ Event Cancelled</h1>
		</div>
		<div class="content">
			<div class="alert alert-danger" style="padding: 15px; background-color: #f8d7da; border-left: 4px solid #dc3545; border-radius: 4px; margin: 20px 0; color: #721c24;">
				<strong style="font-size: 18px;">⚠️ This event has been cancelled</strong>
			</div>
			%s
			<p>Hello,</p>
			<p>We regret to inform you that <strong>%s</strong> has been cancelled and removed from the schedule.</p>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #fff; border: 1px solid #dee2e6; border-radius: 8px;">
				<h3 style="color: #2c3e50;">📋 Event Information</h3>
				<ul style="list-style: none; padding: 0;">
					<li style="margin: 10px 0;"><strong>📌 Event:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>🏢 Organization:</strong> %s</li>
					<li style="margin: 10px 0;"><strong>📅 Created On:</strong> %s</li>
				</ul>
			</div>
			
			<div style="margin: 20px 0; padding: 15px; background-color: #fff3cd; border-left: 4px solid #ffc107; border-radius: 4px;">
				<h3 style="margin-top: 0; color: #856404;">💳 Refund Information</h3>
				<p style="color: #856404; line-height: 1.6;">
					If you have purchased tickets for this event, you will be automatically refunded within 5-7 business days. 
					You will receive a separate confirmation email once the refund is processed.
				</p>
			</div>
			
			<p>For any questions or concerns, please contact our support team.</p>
			
			<p style="text-align: center; margin: 30px 0;">
				<a href="https://ticketly.dpiyumal.me/support" style="display: inline-block; padding: 12px 30px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">
					Contact Support
				</a>
			</p>
		</div>
	`, orgHTML, event.Title, event.Title, organizationName, createdStr)

	html := wrapEventEmailHTML(event.Title, "❌ Event Cancelled", content)

	return email.EmailTemplate{
		Type:    email.EmailEventCancelled,
		Subject: fmt.Sprintf("⚠️ Event Cancelled: %s", event.Title),
		HTML:    html,
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

// wrapEventEmailHTML wraps email content with HTML document structure and styles
func wrapEventEmailHTML(title, headerTitle, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s</title>
	<style>
		body {
			font-family: 'Arial', sans-serif;
			line-height: 1.6;
			color: #333;
			max-width: 600px;
			margin: 0 auto;
			padding: 20px;
			background-color: #f4f4f4;
		}
		.header {
			text-align: center;
			padding: 20px 0;
			border-bottom: 2px solid #eee;
			background-color: #fff;
		}
		.header h1 {
			color: #2c3e50;
			margin: 10px 0;
		}
		.content {
			padding: 20px;
			background-color: #fff;
		}
		.footer {
			text-align: center;
			padding: 20px;
			border-top: 1px solid #eee;
			font-size: 12px;
			color: #777;
			background-color: #fff;
		}
		.alert {
			padding: 15px;
			border-radius: 5px;
			margin: 20px 0;
		}
		.alert-success {
			background-color: #d4edda;
			color: #155724;
			border: 1px solid #c3e6cb;
		}
		.alert-danger {
			background-color: #f8d7da;
			color: #721c24;
			border: 1px solid #f5c6cb;
		}
		.alert-warning {
			background-color: #fff3cd;
			color: #856404;
			border: 1px solid #ffeeba;
		}
		.alert-info {
			background-color: #d1ecf1;
			color: #0c5460;
			border: 1px solid #bee5eb;
		}
		a {
			color: #007bff;
			text-decoration: none;
		}
		a:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	%s
	<div class="footer">
		<p>This is an automated notification from Ticketly.</p>
		<p>&copy; 2025 Ticketly. All rights reserved.</p>
	</div>
</body>
</html>`, title, content)
}
