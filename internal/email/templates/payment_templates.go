package templates

import (
	"fmt"

	"ms-scheduling/internal/email"
	"ms-scheduling/internal/email/builders"
)

// PaymentData represents payment information for email templates
type PaymentData struct {
	PaymentID     string
	OrderID       string
	Amount        float64
	Currency      string
	PaymentMethod string
	TransactionID string
	ProcessedAt   string
	RefundAmount  float64
	RefundReason  string
	EventTitle    string
	SessionTitle  string
}

// GeneratePaymentSuccessEmail generates an email for successful payments
func GeneratePaymentSuccessEmail(payment *PaymentData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#10B981")

	builder.SetHeader("✅ Payment Successful!", "Your payment has been processed")

	builder.AddInfoBox(
		fmt.Sprintf("Payment of <strong>$%.2f</strong> has been successfully processed.", payment.Amount),
		"success",
	)

	details := map[string]string{
		"Payment ID":     payment.PaymentID,
		"Order ID":       payment.OrderID,
		"Amount":         fmt.Sprintf("$%.2f %s", payment.Amount, payment.Currency),
		"Payment Method": payment.PaymentMethod,
		"Transaction ID": payment.TransactionID,
		"Processed At":   payment.ProcessedAt,
	}
	builder.AddDetailsList(details)

	if payment.EventTitle != "" {
		builder.AddSection("🎫 Purchase Details", fmt.Sprintf(`
			<p><strong>Event:</strong> %s</p>
			%s
		`, payment.EventTitle, conditionalSession(payment.SessionTitle)))
	}

	builder.AddDivider()
	builder.AddParagraph("Your tickets are now confirmed and ready to use. You can view them in your account or check your email for the ticket confirmation.")
	// builder.AddButton("View My Tickets", "https://ticketly.com/my-tickets")

	return email.EmailTemplate{
		Type:    email.EmailPaymentSuccess,
		Subject: fmt.Sprintf("Payment Successful - $%.2f", payment.Amount),
		HTML:    builder.Build(),
	}
}

// GeneratePaymentFailedEmail generates an email for failed payments
func GeneratePaymentFailedEmail(payment *PaymentData, reason string) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#EF4444")

	builder.SetHeader("❌ Payment Failed", "We couldn't process your payment")

	builder.AddInfoBox(
		fmt.Sprintf("Unfortunately, your payment of <strong>$%.2f</strong> could not be processed.", payment.Amount),
		"error",
	)

	if reason != "" {
		builder.AddParagraph(fmt.Sprintf("<strong>Reason:</strong> %s", reason))
	}

	details := map[string]string{
		"Payment ID":     payment.PaymentID,
		"Order ID":       payment.OrderID,
		"Amount":         fmt.Sprintf("$%.2f %s", payment.Amount, payment.Currency),
		"Payment Method": payment.PaymentMethod,
		"Attempted At":   payment.ProcessedAt,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("<strong>What to do next:</strong>")
	builder.AddParagraph("• Check your payment method details<br>• Ensure you have sufficient funds<br>• Try a different payment method<br>• Contact your bank if the issue persists")
	// builder.AddButton("Retry Payment", fmt.Sprintf("https://ticketly.com/orders/%s/pay", payment.OrderID))

	return email.EmailTemplate{
		Type:    email.EmailPaymentFailed,
		Subject: "Payment Failed - Action Required",
		HTML:    builder.Build(),
	}
}

// GeneratePaymentPendingEmail generates an email for pending payments
func GeneratePaymentPendingEmail(payment *PaymentData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#F59E0B")

	builder.SetHeader("⏳ Payment Processing", "Your payment is being processed")

	builder.AddInfoBox(
		fmt.Sprintf("Your payment of <strong>$%.2f</strong> is currently being processed.", payment.Amount),
		"warning",
	)

	builder.AddParagraph("This usually takes a few minutes. We'll send you a confirmation email once the payment is complete.")

	details := map[string]string{
		"Payment ID":     payment.PaymentID,
		"Order ID":       payment.OrderID,
		"Amount":         fmt.Sprintf("$%.2f %s", payment.Amount, payment.Currency),
		"Payment Method": payment.PaymentMethod,
		"Initiated At":   payment.ProcessedAt,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("If you don't receive a confirmation within 30 minutes, please contact our support team.")

	return email.EmailTemplate{
		Type:    email.EmailPaymentPending,
		Subject: "Payment Processing - Please Wait",
		HTML:    builder.Build(),
	}
}

// GeneratePaymentRefundedEmail generates an email for refunded payments
func GeneratePaymentRefundedEmail(payment *PaymentData) email.EmailTemplate {
	builder := builders.NewEmailBuilder("Ticketly", "#3B82F6")

	builder.SetHeader("💰 Refund Processed", "Your refund has been issued")

	builder.AddInfoBox(
		fmt.Sprintf("A refund of <strong>$%.2f</strong> has been processed to your original payment method.", payment.RefundAmount),
		"info",
	)

	if payment.RefundReason != "" {
		builder.AddParagraph(fmt.Sprintf("<strong>Reason:</strong> %s", payment.RefundReason))
	}

	details := map[string]string{
		"Payment ID":     payment.PaymentID,
		"Order ID":       payment.OrderID,
		"Refund Amount":  fmt.Sprintf("$%.2f %s", payment.RefundAmount, payment.Currency),
		"Payment Method": payment.PaymentMethod,
		"Processed At":   payment.ProcessedAt,
	}
	builder.AddDetailsList(details)

	builder.AddDivider()
	builder.AddParagraph("<strong>When will I receive my refund?</strong>")
	builder.AddParagraph("Refunds typically appear in your account within 5-7 business days, depending on your bank or payment provider.")
	builder.AddParagraph("If you have any questions, please don't hesitate to contact our support team.")

	return email.EmailTemplate{
		Type:    email.EmailPaymentRefunded,
		Subject: fmt.Sprintf("Refund Processed - $%.2f", payment.RefundAmount),
		HTML:    builder.Build(),
	}
}

// Helper functions

func conditionalSession(sessionTitle string) string {
	if sessionTitle == "" {
		return ""
	}
	return fmt.Sprintf("<p><strong>Session:</strong> %s</p>", sessionTitle)
}
