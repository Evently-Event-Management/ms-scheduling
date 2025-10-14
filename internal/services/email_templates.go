package services

import "fmt"

// EmailTemplateType defines the type of email template to use
type EmailTemplateType string

const (
	OrderConfirmed  EmailTemplateType = "ORDER_CONFIRMED"
	OrderPending    EmailTemplateType = "ORDER_PENDING"
	OrderCancelled  EmailTemplateType = "ORDER_CANCELLED"
	OrderProcessing EmailTemplateType = "ORDER_PROCESSING"
)

// Helper functions
func generateDiscountHTML(order *OrderCreatedEvent) string {
	if order.DiscountAmount > 0 {
		return fmt.Sprintf("<div><strong>Discount:</strong> %s ($%.2f)</div>", order.DiscountCode, order.DiscountAmount)
	}
	return ""
}

func generatePaymentTimeHTML(order *OrderCreatedEvent) string {
	if order.PaymentAT != "" {
		return fmt.Sprintf("<div><strong>Payment Time:</strong> %s</div>", order.PaymentAT)
	}
	return ""
}

// GetEmailSubject returns the appropriate email subject based on template type
func GetEmailSubject(templateType EmailTemplateType, orderID string) string {
	switch templateType {
	case OrderConfirmed:
		return fmt.Sprintf("Order Confirmed - %s", orderID)
	case OrderPending:
		return fmt.Sprintf("Order Pending Payment - %s", orderID)
	case OrderCancelled:
		return fmt.Sprintf("Order Cancelled - %s", orderID)
	case OrderProcessing:
		return fmt.Sprintf("Order Processing - %s", orderID)
	default:
		return fmt.Sprintf("Order Update - %s", orderID)
	}
}

// GenerateHTMLEmailTemplate generates HTML email content based on order status
func GenerateHTMLEmailTemplate(templateType EmailTemplateType, order *OrderCreatedEvent) string {
	// Common CSS styles for all email templates
	const styles = `
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
			.header h1 {
				color: #2c3e50;
				margin: 0;
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
			.order-details {
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
				color: white;
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
		</style>
	`

	// Generate ticket list HTML
	ticketListHTML := ""
	for _, ticket := range order.Tickets {
		ticketListHTML += fmt.Sprintf(`
			<div class="ticket-item">
				<div><span class="color-swatch" style="background-color: %s"></span> <strong>%s</strong> (%s)</div>
				<div>Seat: %s</div>
				<div>Price: $%.2f</div>
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
			<div><strong>Subtotal:</strong> $%.2f</div>
			%s
			<div><strong>Total Price:</strong> $%.2f</div>
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

	var content string
	switch templateType {
	case OrderConfirmed:
		content = fmt.Sprintf(`
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
				<a href="#" class="btn btn-primary">View My Tickets</a>
			</div>
			<div class="footer">
				<p>This is an automated email. Please do not reply.</p>
				<p>&copy; 2025 Ticketly. All rights reserved.</p>
			</div>
		`, orderDetailsHTML, ticketListHTML)

	case OrderPending:
		content = fmt.Sprintf(`
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
				<a href="#" class="btn btn-primary">Complete Payment Now</a>
			</div>
			<div class="footer">
				<p>This is an automated email. Please do not reply.</p>
				<p>&copy; 2025 Ticketly. All rights reserved.</p>
			</div>
		`, orderDetailsHTML, ticketListHTML)

	case OrderCancelled:
		content = fmt.Sprintf(`
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
				<a href="#" class="btn btn-primary">Browse Events</a>
			</div>
			<div class="footer">
				<p>This is an automated email. Please do not reply.</p>
				<p>&copy; 2025 Ticketly. All rights reserved.</p>
			</div>
		`, orderDetailsHTML, ticketListHTML)

	case OrderProcessing:
		content = fmt.Sprintf(`
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

	default:
		// Fallback to a generic template
		content = fmt.Sprintf(`
			<div class="header">
				<h1>Order Update</h1>
			</div>
			<div class="content">
				<p>Dear Customer,</p>
				<p>There has been an update to your order.</p>
				%s
				<h3>Ticket Details:</h3>
				<div class="ticket-list">
					%s
				</div>
				<p>If you have any questions, please contact our support team.</p>
			</div>
			<div class="footer">
				<p>This is an automated email. Please do not reply.</p>
				<p>&copy; 2025 Ticketly. All rights reserved.</p>
			</div>
		`, orderDetailsHTML, ticketListHTML)
	}

	// Combine styles with content
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
	`, GetEmailSubject(templateType, order.OrderID), styles, content)
}
