package templates

import (
	"fmt"
	"strings"

	"ms-scheduling/internal/email"
	"ms-scheduling/internal/email/builders"
)

// OrderData represents order information for email templates
type OrderData struct {
	OrderID        string
	UserID         string
	EventID        string
	SessionID      string
	OrganizationID string
	Status         string
	SubTotal       float64
	DiscountCode   string
	DiscountAmount float64
	Price          float64
	CreatedAt      string
	PaymentAt      string
	Tickets        []TicketData
	EventTitle     string
	SessionTitle   string
}

type TicketData struct {
	TicketID        string
	SeatLabel       string
	TierName        string
	PriceAtPurchase float64
}

// GenerateOrderConfirmedEmail generates an email for confirmed orders
func GenerateOrderConfirmedEmail(order *OrderData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#10B981")

	builder.SetHeader("✅ Order Confirmed!", "Your order has been successfully processed")

	builder.AddInfoBox(
		fmt.Sprintf("Thank you for your purchase! Your order <strong>#%s</strong> has been confirmed.", order.OrderID),
		"success",
	)

	// Order summary
	builder.AddSection("📦 Order Summary", buildOrderSummary(order))

	// Tickets
	if len(order.Tickets) > 0 {
		builder.AddSection("🎫 Your Tickets", buildTicketList(order.Tickets))
	}

	// Payment details
	builder.AddSection("💳 Payment Details", buildPaymentSummary(order))

	builder.AddDivider()
	builder.AddParagraph("Your tickets have been sent to your email and are also available in your account.")
	// builder.AddButton("View My Tickets", "https://ticketly.com/my-tickets")

	return email.EmailTemplate{
		Type:    email.EmailOrderConfirmed,
		Subject: fmt.Sprintf("Order Confirmed - #%s", order.OrderID),
		HTML:    builder.Build(),
	}
}

// GenerateOrderPendingEmail generates an email for pending orders
func GenerateOrderPendingEmail(order *OrderData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#F59E0B")

	builder.SetHeader("⏳ Order Pending Payment", "Complete your payment to confirm your order")

	builder.AddInfoBox(
		fmt.Sprintf("Your order <strong>#%s</strong> is waiting for payment confirmation.", order.OrderID),
		"warning",
	)

	builder.AddParagraph("Your tickets are reserved, but the order is not yet complete. Please complete your payment to confirm the purchase.")

	// Order summary
	builder.AddSection("📦 Order Summary", buildOrderSummary(order))

	// Tickets
	if len(order.Tickets) > 0 {
		builder.AddSection("🎫 Reserved Tickets", buildTicketList(order.Tickets))
	}

	// Payment details
	builder.AddSection("💳 Amount Due", fmt.Sprintf(`
		<p style="font-size: 24px; font-weight: bold; color: #F59E0B;">$%.2f</p>
	`, order.Price))

	builder.AddDivider()
	builder.AddParagraph("⚠️ <strong>Important:</strong> Your tickets are reserved for a limited time. Please complete payment soon to avoid losing your reservation.")
	// builder.AddButton("Complete Payment", fmt.Sprintf("https://ticketly.com/orders/%s/pay", order.OrderID))

	return email.EmailTemplate{
		Type:    email.EmailOrderPending,
		Subject: fmt.Sprintf("Payment Pending - Order #%s", order.OrderID),
		HTML:    builder.Build(),
	}
}

// GenerateOrderCancelledEmail generates an email for cancelled orders
func GenerateOrderCancelledEmail(order *OrderData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#EF4444")

	builder.SetHeader("❌ Order Cancelled", "Your order has been cancelled")

	builder.AddInfoBox(
		fmt.Sprintf("Order <strong>#%s</strong> has been cancelled.", order.OrderID),
		"error",
	)

	builder.AddParagraph("This order has been cancelled and your tickets are no longer valid.")

	// Order summary
	builder.AddSection("📦 Cancelled Order Details", buildOrderSummary(order))

	builder.AddDivider()
	builder.AddParagraph("<strong>Refund Information:</strong>")
	builder.AddParagraph("If you were charged for this order, a refund will be processed within 5-7 business days. You will receive a confirmation email once the refund is complete.")

	return email.EmailTemplate{
		Type:    email.EmailOrderCancelled,
		Subject: fmt.Sprintf("Order Cancelled - #%s", order.OrderID),
		HTML:    builder.Build(),
	}
}

// GenerateOrderUpdatedEmail generates an email for order updates
func GenerateOrderUpdatedEmail(order *OrderData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#4F46E5")

	builder.SetHeader("📝 Order Update", "Your order has been updated")

	builder.AddInfoBox(
		fmt.Sprintf("Order <strong>#%s</strong> has been updated.", order.OrderID),
		"info",
	)

	// Order summary
	builder.AddSection("📦 Order Details", buildOrderSummary(order))

	// Tickets
	if len(order.Tickets) > 0 {
		builder.AddSection("🎫 Your Tickets", buildTicketList(order.Tickets))
	}

	return email.EmailTemplate{
		Type:    email.EmailOrderUpdated,
		Subject: fmt.Sprintf("Order Updated - #%s", order.OrderID),
		HTML:    builder.Build(),
	}
}

// Helper functions

func buildOrderSummary(order *OrderData) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("<p><strong>Order ID:</strong> %s</p>", order.OrderID))

	if order.EventTitle != "" {
		summary.WriteString(fmt.Sprintf("<p><strong>Event:</strong> %s</p>", order.EventTitle))
	}

	if order.SessionTitle != "" {
		summary.WriteString(fmt.Sprintf("<p><strong>Session:</strong> %s</p>", order.SessionTitle))
	}

	summary.WriteString(fmt.Sprintf("<p><strong>Order Date:</strong> %s</p>", order.CreatedAt))
	summary.WriteString(fmt.Sprintf("<p><strong>Status:</strong> <span style='color: %s; font-weight: bold;'>%s</span></p>",
		getStatusColor(order.Status), strings.ToUpper(order.Status)))

	return summary.String()
}

func buildTicketList(tickets []TicketData) string {
	var list strings.Builder

	list.WriteString(`<div style="background-color: #F9FAFB; border-radius: 8px; padding: 15px;">`)

	for i, ticket := range tickets {
		list.WriteString(fmt.Sprintf(`
			<div style="background-color: white; border-radius: 6px; padding: 12px; margin-bottom: 10px; border-left: 4px solid #4F46E5;">
				<p style="margin: 0; font-weight: bold; color: #1F2937;">Ticket %d: %s</p>
				<p style="margin: 5px 0 0 0; color: #6B7280; font-size: 14px;">
					Seat: %s | Tier: %s | Price: $%.2f
				</p>
			</div>
		`, i+1, ticket.TicketID[:8]+"...", ticket.SeatLabel, ticket.TierName, ticket.PriceAtPurchase))
	}

	list.WriteString("</div>")
	return list.String()
}

func buildPaymentSummary(order *OrderData) string {
	var summary strings.Builder

	summary.WriteString(`<table style="width: 100%; border-collapse: collapse;">`)

	// Subtotal
	summary.WriteString(fmt.Sprintf(`
		<tr>
			<td style="padding: 8px 0; color: #4B5563;">Subtotal:</td>
			<td style="padding: 8px 0; text-align: right; color: #1F2937;">$%.2f</td>
		</tr>
	`, order.SubTotal))

	// Discount
	if order.DiscountAmount > 0 {
		summary.WriteString(fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #10B981;">Discount (%s):</td>
				<td style="padding: 8px 0; text-align: right; color: #10B981;">-$%.2f</td>
			</tr>
		`, order.DiscountCode, order.DiscountAmount))
	}

	// Total
	summary.WriteString(fmt.Sprintf(`
		<tr style="border-top: 2px solid #E5E7EB;">
			<td style="padding: 12px 0; font-size: 18px; font-weight: bold; color: #1F2937;">Total:</td>
			<td style="padding: 12px 0; text-align: right; font-size: 18px; font-weight: bold; color: #1F2937;">$%.2f</td>
		</tr>
	`, order.Price))

	if order.PaymentAt != "" {
		summary.WriteString(fmt.Sprintf(`
			<tr>
				<td colspan="2" style="padding-top: 8px; color: #6B7280; font-size: 14px;">
					Paid on: %s
				</td>
			</tr>
		`, order.PaymentAt))
	}

	summary.WriteString("</table>")
	return summary.String()
}

func getStatusColor(status string) string {
	switch strings.ToLower(status) {
	case "completed", "confirmed":
		return "#10B981"
	case "pending":
		return "#F59E0B"
	case "cancelled", "failed":
		return "#EF4444"
	case "processing":
		return "#3B82F6"
	default:
		return "#6B7280"
	}
}
