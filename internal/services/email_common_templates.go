package services

import (
	"fmt"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
)

// EmailType defines the type of email to be sent
type EmailType string

// Email template types
const (
	// Order related emails
	EmailOrderConfirmed  EmailType = "ORDER_CONFIRMED"
	EmailOrderPending    EmailType = "ORDER_PENDING"
	EmailOrderCancelled  EmailType = "ORDER_CANCELLED"
	EmailOrderProcessing EmailType = "ORDER_PROCESSING"

	// Session related emails
	EmailSessionReminder      EmailType = "SESSION_REMINDER"
	EmailSessionStartReminder EmailType = "SESSION_START_REMINDER"
	EmailSessionSalesReminder EmailType = "SESSION_SALES_REMINDER"
	EmailSessionUpdate        EmailType = "SESSION_UPDATE"
	EmailSessionCreation      EmailType = "SESSION_CREATION"
	EmailSessionCancellation  EmailType = "SESSION_CANCELLATION"

	// Event related emails
	EmailEventUpdate   EmailType = "EVENT_UPDATE"
	EmailEventCreation EmailType = "EVENT_CREATION"
)

// Common CSS styles for all emails
const commonStyles = `
<style>
	body {
		font-family: 'Arial', sans-serif;
		line-height: 1.6;
		color: #333;
		max-width: 600px;
		margin: 0 auto;
		padding: 20px;
	}
	.header {
		text-align: center;
		padding: 20px 0;
		border-bottom: 2px solid #eee;
	}
	.header img {
		max-width: 200px;
		height: auto;
	}
	.header h1 {
		color: #2c3e50;
		margin: 10px 0;
	}
	.content {
		padding: 20px 0;
	}
	.footer {
		text-align: center;
		padding-top: 20px;
		border-top: 1px solid #eee;
		font-size: 12px;
		color: #777;
	}
	.order-details, .session-details, .event-details {
		background-color: #f9f9f9;
		border-radius: 5px;
		padding: 15px;
		margin: 20px 0;
	}
	.ticket-list {
		margin: 20px 0;
	}
	.ticket-item {
		border-left: 4px solid #ddd;
		padding: 10px 15px;
		margin-bottom: 10px;
		background-color: #f9f9f9;
		border-radius: 0 5px 5px 0;
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
	.alert-warning {
		background-color: #fff3cd;
		color: #856404;
		border: 1px solid #ffeeba;
	}
	.alert-danger {
		background-color: #f8d7da;
		color: #721c24;
		border: 1px solid #f5c6cb;
	}
	.alert-info {
		background-color: #d1ecf1;
		color: #0c5460;
		border: 1px solid #bee5eb;
	}
	.btn {
		display: inline-block;
		padding: 10px 20px;
		margin: 10px 0;
		font-size: 16px;
		font-weight: bold;
		text-align: center;
		text-decoration: none;
		border-radius: 5px;
		cursor: pointer;
	}
	.btn-primary {
		background-color: #007bff;
		color: white !important;
		border: none;
	}
	.btn-success {
		background-color: #28a745;
		color: white !important;
		border: none;
	}
	.btn-danger {
		background-color: #dc3545;
		color: white !important;
		border: none;
	}
	.btn-warning {
		background-color: #ffc107;
		color: #212529 !important;
		border: none;
	}
	.color-swatch {
		display: inline-block;
		width: 15px;
		height: 15px;
		margin-right: 5px;
		border-radius: 3px;
		vertical-align: middle;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	table th, table td {
		padding: 8px;
		text-align: left;
		border-bottom: 1px solid #ddd;
	}
	.text-center {
		text-align: center;
	}
	.text-right {
		text-align: right;
	}
	.mt-3 {
		margin-top: 15px;
	}
	.mt-4 {
		margin-top: 20px;
	}
	.mb-3 {
		margin-bottom: 15px;
	}
	.mb-4 {
		margin-bottom: 20px;
	}
	a {
		color: #007bff;
		text-decoration: none;
	}
	a:hover {
		text-decoration: underline;
	}
</style>
`

// Helper functions to generate URLs based on config
func generateEventURL(cfg *config.Config, eventID string) string {
	return fmt.Sprintf("%s/events/%s", cfg.FrontendURL, eventID)
}

func generateSessionURL(cfg *config.Config, eventID, sessionID string) string {
	return fmt.Sprintf("%s/events/%s/%s", cfg.FrontendURL, eventID, sessionID)
}

func generateOrderURL(cfg *config.Config, orderID string) string {
	return fmt.Sprintf("%s/orders/%s", cfg.FrontendURL, orderID)
}

func generatePaymentSuccessURL(cfg *config.Config) string {
	return fmt.Sprintf("%s/payment-success", cfg.FrontendURL)
}

func generateOrdersListURL(cfg *config.Config) string {
	return fmt.Sprintf("%s/orders", cfg.FrontendURL)
}

func generateEventsListURL(cfg *config.Config) string {
	return fmt.Sprintf("%s/events", cfg.FrontendURL)
}

func generateUnsubscribeURL(cfg *config.Config, subscriptionID string) string {
	return fmt.Sprintf("%s/unsubscribe/%s", cfg.FrontendURL, subscriptionID)
}

func generateVenueHTML(venue string) string {
	if venue != "" {
		return fmt.Sprintf("<li><strong>Venue:</strong> %s</li>", venue)
	}
	return ""
}

// EmailTemplate holds the structure for an email
type EmailTemplate struct {
	Subject string
	HTML    string
}

// GenerateEmailTemplate creates an email template based on the template type
func GenerateEmailTemplate(cfg *config.Config, emailType EmailType, data interface{}) EmailTemplate {
	switch emailType {
	case EmailOrderConfirmed:
		return generateOrderConfirmedEmail(cfg, data.(*OrderCreatedEvent))
	case EmailOrderPending:
		return generateOrderPendingEmail(cfg, data.(*OrderCreatedEvent))
	case EmailOrderCancelled:
		return generateOrderCancelledEmail(cfg, data.(*OrderCreatedEvent))
	case EmailOrderProcessing:
		return generateOrderProcessingEmail(cfg, data.(*OrderCreatedEvent))
	case EmailSessionStartReminder:
		return generateSessionStartReminderEmail(cfg, data.(*SessionReminderInfo))
	case EmailSessionSalesReminder:
		return generateSessionSalesReminderEmail(cfg, data.(*SessionReminderInfo))
	// Add other email templates as needed
	default:
		return EmailTemplate{
			Subject: "Ticketly Notification",
			HTML:    generateDefaultEmail(cfg),
		}
	}
}

// Generate HTML document with content
func wrapInHTMLDocument(title string, content string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s</title>
	%s
</head>
<body>
	%s
</body>
</html>
	`, title, commonStyles, content)
}

// Default email
func generateDefaultEmail(cfg *config.Config) string {
	content := `
<div class="header">
	<h1>Ticketly Notification</h1>
</div>
<div class="content">
	<p>This is a notification from Ticketly.</p>
</div>
<div class="footer">
	<p>This is an automated email. Please do not reply.</p>
	<p>&copy; 2025 Ticketly. All rights reserved.</p>
</div>
	`
	return wrapInHTMLDocument("Ticketly Notification", content)
}

// Order confirmed email
func generateOrderConfirmedEmail(cfg *config.Config, order *OrderCreatedEvent) EmailTemplate {
	subject := fmt.Sprintf("Order Confirmed - %s", order.OrderID)

	// Generate ticket list HTML
	ticketListHTML := ""
	for _, ticket := range order.Tickets {
		ticketListHTML += fmt.Sprintf(`
			<div class="ticket-item">
				<div><span class="color-swatch" style="background-color: %s"></span> <strong>%s</strong> (%s)</div>
				<div>Seat: %s</div>
				<div>Price: LKR%.2f</div>
			</div>
		`, ticket.Colour, ticket.TierName, ticket.TierID, ticket.SeatLabel, ticket.PriceAtPurchase)
	}

	// Order details section
	orderDetailsHTML := fmt.Sprintf(`
		<div class="order-details">
			<div><strong>Order ID:</strong> %s</div>
			<div><strong>Event ID:</strong> %s</div>
			<div><strong>Session ID:</strong> %s</div>
			<div><strong>Status:</strong> %s</div>
			<div><strong>Subtotal:</strong> LKR%.2f</div>
			%s
			<div><strong>Total Price:</strong> LKR%.2f</div>
			<div><strong>Created At:</strong> %s</div>
			%s
		</div>
	`,
		order.OrderID,
		order.EventID,
		order.SessionID,
		order.Status,
		order.SubTotal,
		generateDiscountHTML(order),
		order.Price,
		order.CreatedAt,
		generatePaymentTimeHTML(order))

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Order Confirmed</h1>
		</div>
		<div class="content">
			<div class="alert alert-success">
				Your payment has been successfully processed and your order is confirmed.
			</div>
			<p>Dear Customer,</p>
			<p>Thank you for your purchase! Your order has been confirmed and your tickets are ready.</p>
			%s
			<h3>Your Tickets:</h3>
			<div class="ticket-list">
				%s
			</div>
			<p>Please keep this email for your records. You'll need to show your tickets when you arrive at the event.</p>
			<p>We look forward to seeing you there!</p>
			<p>
				<a href="%s" class="btn btn-primary">View My Tickets</a>
				<a href="%s" class="btn btn-success">Browse More Events</a>
			</p>
		</div>
		<div class="footer">
			<p>This is an automated email. Please do not reply.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`, orderDetailsHTML, ticketListHTML,
		generateOrderURL(cfg, order.OrderID),
		generateEventsListURL(cfg))

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Order pending email
func generateOrderPendingEmail(cfg *config.Config, order *OrderCreatedEvent) EmailTemplate {
	subject := fmt.Sprintf("Order Pending Payment - %s", order.OrderID)

	// Generate ticket list HTML
	ticketListHTML := ""
	for _, ticket := range order.Tickets {
		ticketListHTML += fmt.Sprintf(`
			<div class="ticket-item">
				<div><span class="color-swatch" style="background-color: %s"></span> <strong>%s</strong> (%s)</div>
				<div>Seat: %s</div>
				<div>Price: LKR%.2f</div>
			</div>
		`, ticket.Colour, ticket.TierName, ticket.TierID, ticket.SeatLabel, ticket.PriceAtPurchase)
	}

	// Order details section
	orderDetailsHTML := fmt.Sprintf(`
		<div class="order-details">
			<div><strong>Order ID:</strong> %s</div>
			<div><strong>Event ID:</strong> %s</div>
			<div><strong>Session ID:</strong> %s</div>
			<div><strong>Status:</strong> %s</div>
			<div><strong>Subtotal:</strong> LKR%.2f</div>
			%s
			<div><strong>Total Price:</strong> LKR%.2f</div>
			<div><strong>Created At:</strong> %s</div>
		</div>
	`,
		order.OrderID,
		order.EventID,
		order.SessionID,
		order.Status,
		order.SubTotal,
		generateDiscountHTML(order),
		order.Price,
		order.CreatedAt)

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Payment Required</h1>
		</div>
		<div class="content">
			<div class="alert alert-warning">
				Your order is pending payment. Please complete your payment to secure your tickets.
			</div>
			<p>Dear Customer,</p>
			<p>We've received your order, but payment is still required to confirm your tickets.</p>
			%s
			<h3>Selected Tickets:</h3>
			<div class="ticket-list">
				%s
			</div>
			<p><strong>Important:</strong> Your tickets are reserved for a limited time. Please complete payment within the next 15 minutes to avoid losing your reservation.</p>
			<p>
				<a href="%s" class="btn btn-primary">Complete Payment Now</a>
			</p>
		</div>
		<div class="footer">
			<p>This is an automated email. Please do not reply.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`, orderDetailsHTML, ticketListHTML,
		generateOrderURL(cfg, order.OrderID))

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Order cancelled email
func generateOrderCancelledEmail(cfg *config.Config, order *OrderCreatedEvent) EmailTemplate {
	subject := fmt.Sprintf("Order Cancelled - %s", order.OrderID)

	// Generate ticket list HTML
	ticketListHTML := ""
	for _, ticket := range order.Tickets {
		ticketListHTML += fmt.Sprintf(`
			<div class="ticket-item">
				<div><span class="color-swatch" style="background-color: %s"></span> <strong>%s</strong> (%s)</div>
				<div>Seat: %s</div>
				<div>Price: LKR%.2f</div>
			</div>
		`, ticket.Colour, ticket.TierName, ticket.TierID, ticket.SeatLabel, ticket.PriceAtPurchase)
	}

	// Order details section
	orderDetailsHTML := fmt.Sprintf(`
		<div class="order-details">
			<div><strong>Order ID:</strong> %s</div>
			<div><strong>Event ID:</strong> %s</div>
			<div><strong>Session ID:</strong> %s</div>
			<div><strong>Status:</strong> %s</div>
			<div><strong>Subtotal:</strong> LKR%.2f</div>
			%s
			<div><strong>Total Price:</strong> LKR%.2f</div>
			<div><strong>Created At:</strong> %s</div>
			%s
		</div>
	`,
		order.OrderID,
		order.EventID,
		order.SessionID,
		order.Status,
		order.SubTotal,
		generateDiscountHTML(order),
		order.Price,
		order.CreatedAt,
		generatePaymentTimeHTML(order))

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Order Cancelled</h1>
		</div>
		<div class="content">
			<div class="alert alert-danger">
				Your order has been cancelled. No payment has been processed.
			</div>
			<p>Dear Customer,</p>
			<p>We're sorry to inform you that your order has been cancelled. This could be due to payment timeout, payment failure, or as requested by you.</p>
			%s
			<h3>Tickets (Not Reserved):</h3>
			<div class="ticket-list">
				%s
			</div>
			<p>If you still wish to attend this event, please make a new purchase through our website.</p>
			<p>If you believe this cancellation was made in error, please contact our support team.</p>
			<p>
				<a href="%s" class="btn btn-primary">Browse Events</a>
			</p>
		</div>
		<div class="footer">
			<p>This is an automated email. Please do not reply.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`, orderDetailsHTML, ticketListHTML,
		generateEventsListURL(cfg))

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Order processing email
func generateOrderProcessingEmail(cfg *config.Config, order *OrderCreatedEvent) EmailTemplate {
	subject := fmt.Sprintf("Order Processing - %s", order.OrderID)

	// Generate ticket list HTML
	ticketListHTML := ""
	for _, ticket := range order.Tickets {
		ticketListHTML += fmt.Sprintf(`
			<div class="ticket-item">
				<div><span class="color-swatch" style="background-color: %s"></span> <strong>%s</strong> (%s)</div>
				<div>Seat: %s</div>
				<div>Price: LKR%.2f</div>
			</div>
		`, ticket.Colour, ticket.TierName, ticket.TierID, ticket.SeatLabel, ticket.PriceAtPurchase)
	}

	// Order details section
	orderDetailsHTML := fmt.Sprintf(`
		<div class="order-details">
			<div><strong>Order ID:</strong> %s</div>
			<div><strong>Event ID:</strong> %s</div>
			<div><strong>Session ID:</strong> %s</div>
			<div><strong>Status:</strong> %s</div>
			<div><strong>Subtotal:</strong> LKR%.2f</div>
			%s
			<div><strong>Total Price:</strong> LKR%.2f</div>
			<div><strong>Created At:</strong> %s</div>
			%s
		</div>
	`,
		order.OrderID,
		order.EventID,
		order.SessionID,
		order.Status,
		order.SubTotal,
		generateDiscountHTML(order),
		order.Price,
		order.CreatedAt,
		generatePaymentTimeHTML(order))

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Order Processing</h1>
		</div>
		<div class="content">
			<div class="alert alert-info">
				Your payment is being processed. We'll notify you once it's complete.
			</div>
			<p>Dear Customer,</p>
			<p>We've received your payment and it's currently being processed. This usually takes just a few moments.</p>
			%s
			<h3>Your Tickets (Processing):</h3>
			<div class="ticket-list">
				%s
			</div>
			<p>You'll receive a confirmation email once your payment has been successfully processed.</p>
			<p>No further action is required from you at this time.</p>
		</div>
		<div class="footer">
			<p>This is an automated email. Please do not reply.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`, orderDetailsHTML, ticketListHTML)

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Session start reminder email
func generateSessionStartReminderEmail(cfg *config.Config, sessionInfo *SessionReminderInfo) EmailTemplate {
	// Convert timestamps to readable format
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)
	endTime := models.MicroTimestampToTime(sessionInfo.EndTime)

	// Format date and time more user-friendly
	dateStr := startTime.Format("Monday, January 2, 2006")
	startTimeStr := startTime.Format("3:04 PM")
	endTimeStr := endTime.Format("3:04 PM")

	// Calculate duration
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

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Your Event"
	}

	subject := fmt.Sprintf("🔔 Reminder: %s is tomorrow!", eventTitle)

	// Generate calendar links
	googleCalLink := fmt.Sprintf("https://calendar.google.com/calendar/render?action=TEMPLATE&text=%s&dates=%s/%s&details=%s&location=%s",
		urlEscape(eventTitle),
		startTime.Format("20060102T150405"),
		endTime.Format("20060102T150405"),
		urlEscape(eventTitle),
		urlEscape(sessionInfo.VenueDetails))

	appleCalLink := fmt.Sprintf("%s/calendar/event-%s.ics", cfg.FrontendURL, sessionInfo.SessionID)

	sessionURL := generateSessionURL(cfg, sessionInfo.EventID, sessionInfo.SessionID)

	// Generate venue HTML if available
	var venueHTML string
	if sessionInfo.VenueDetails != "" {
		venueHTML = fmt.Sprintf("<li><strong>Venue:</strong> %s</li>", sessionInfo.VenueDetails)
	} else {
		venueHTML = ""
	}

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Event Reminder</h1>
		</div>
		<div class="content">
			<div class="alert alert-info">
				<strong>%s</strong> is happening tomorrow!
			</div>
			<p>Hello,</p>
			<p>This is a friendly reminder about your upcoming event tomorrow.</p>
			
			<div class="session-details">
				<h3>📅 Event Details:</h3>
				<ul>
					<li><strong>Event:</strong> %s</li>
					<li><strong>Date:</strong> %s</li>
					<li><strong>Time:</strong> %s to %s</li>
					<li><strong>Duration:</strong> %s</li>
					%s
					<li><strong>Status:</strong> %s</li>
				</ul>
			</div>
			
			<p class="text-center mt-4">
				<a href="%s" class="btn btn-primary">View Event Details</a>
			</p>
			
			<div class="mt-4">
				<h3>📱 Add to Calendar:</h3>
				<p>
					<a href="%s" target="_blank">Add to Google Calendar</a> | 
					<a href="%s" target="_blank">Add to Apple Calendar</a>
				</p>
			</div>
			
			<div class="mt-4">
				<h4>📋 Pre-Event Checklist:</h4>
				<ul>
					<li>Plan your route to the venue</li>
					<li>Have your tickets ready</li>
					<li>Check weather conditions</li>
					<li>Arrive early to find good parking</li>
				</ul>
			</div>
			
			<p>We look forward to seeing you tomorrow!</p>
		</div>
		<div class="footer">
			<p>This is an automated reminder. Please do not reply to this email.</p>
			<p><a href="%s">Unsubscribe</a> from these notifications.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`,
		eventTitle,
		eventTitle,
		dateStr,
		startTimeStr,
		endTimeStr,
		durationStr,
		venueHTML,
		sessionInfo.Status,
		sessionURL,
		googleCalLink,
		appleCalLink,
		generateUnsubscribeURL(cfg, sessionInfo.SessionID))

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Session sales reminder email
func generateSessionSalesReminderEmail(cfg *config.Config, sessionInfo *SessionReminderInfo) EmailTemplate {
	// Convert timestamps to readable format
	salesStartTime := models.MicroTimestampToTime(sessionInfo.SalesStartTime)
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)

	// Format date and time more user-friendly
	salesDateStr := salesStartTime.Format("Monday, January 2, 2006")
	salesTimeStr := salesStartTime.Format("3:04 PM")
	eventDateStr := startTime.Format("Monday, January 2, 2006")

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Event"
	}

	subject := fmt.Sprintf("🎟️ Tickets for %s will be available soon!", eventTitle)

	sessionURL := generateSessionURL(cfg, sessionInfo.EventID, sessionInfo.SessionID)

	// Generate venue HTML if available
	var venueHTML string
	if sessionInfo.VenueDetails != "" {
		venueHTML = fmt.Sprintf("<li><strong>Venue:</strong> %s</li>", sessionInfo.VenueDetails)
	} else {
		venueHTML = ""
	}

	content := fmt.Sprintf(`
		<div class="header">
			<h1>Tickets Available Soon!</h1>
		</div>
		<div class="content">
			<div class="alert alert-warning">
				<strong>Tickets for %s will be available in 30 minutes!</strong>
			</div>
			<p>Hello,</p>
			<p>Don't miss your chance to secure your spot for this event. Tickets will be available for purchase shortly.</p>
			
			<div class="session-details">
				<h3>🎫 Ticket Sales Information:</h3>
				<ul>
					<li><strong>Sales Start:</strong> %s at %s</li>
					<li><strong>Event Date:</strong> %s</li>
					<li><strong>Event Title:</strong> %s</li>
					%s
				</ul>
			</div>
			
			<p class="text-center mt-4">
				<a href="%s" class="btn btn-primary">Buy Tickets When Available</a>
			</p>
			
			<p class="mt-4">
				<strong>Tips for Quick Purchase:</strong>
				<ul>
					<li>Sign in to your account before sales begin</li>
					<li>Have your payment method ready</li>
					<li>Check that your billing information is up to date</li>
				</ul>
			</p>
			
			<p>Be ready to purchase as soon as tickets are available!</p>
		</div>
		<div class="footer">
			<p>This is an automated notification. Please do not reply to this email.</p>
			<p><a href="%s">Unsubscribe</a> from these notifications.</p>
			<p>&copy; 2025 Ticketly. All rights reserved.</p>
		</div>
	`,
		eventTitle,
		salesDateStr,
		salesTimeStr,
		eventDateStr,
		eventTitle,
		venueHTML,
		sessionURL,
		generateUnsubscribeURL(cfg, sessionInfo.SessionID))

	return EmailTemplate{
		Subject: subject,
		HTML:    wrapInHTMLDocument(subject, content),
	}
}

// Helper function for URL escaping
func urlEscape(s string) string {
	return s // This is a placeholder - in production code, you would use url.QueryEscape
}
