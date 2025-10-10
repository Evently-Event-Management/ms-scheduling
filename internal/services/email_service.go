package services

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type EmailService struct {
	SMTPHost  string
	SMTPPort  string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

func NewEmailService(smtpHost, smtpPort, username, password, fromEmail, fromName string) *EmailService {
	return &EmailService{
		SMTPHost:  smtpHost,
		SMTPPort:  smtpPort,
		Username:  username,
		Password:  password,
		FromEmail: fromEmail,
		FromName:  fromName,
	}
}

// SendEmail sends an email using SMTP
func (e *EmailService) SendEmail(to, subject, body string) error {
	// SMTP server configuration
	smtpServer := fmt.Sprintf("%s:%s", e.SMTPHost, e.SMTPPort)

	// Authentication
	auth := smtp.PlainAuth("", e.Username, e.Password, e.SMTPHost)

	// Email headers
	from := fmt.Sprintf("%s <%s>", e.FromName, e.FromEmail)

	// Compose message
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		from, to, subject, e.formatEmailBody(body)))

	// Send email
	err := smtp.SendMail(smtpServer, auth, e.FromEmail, []string{to}, msg)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("Email sent successfully to %s", to)
	return nil
}

// formatEmailBody formats the email body as HTML
func (e *EmailService) formatEmailBody(body string) string {
	// Convert plain text to HTML
	htmlBody := strings.ReplaceAll(body, "\n", "<br>")

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Order Confirmation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .content { background-color: #ffffff; padding: 20px; border: 1px solid #dee2e6; border-radius: 5px; }
        .footer { margin-top: 20px; text-align: center; color: #6c757d; font-size: 14px; }
        .ticket-item { background-color: #f8f9fa; padding: 10px; margin: 5px 0; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Ticketly - Order Confirmation</h2>
        </div>
        <div class="content">
            %s
        </div>
        <div class="footer">
            <p>This is an automated email. Please do not reply.</p>
            <p>&copy; 2025 Ticketly. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, htmlBody)
}

// SendOrderConfirmationEmail sends a formatted order confirmation email
func (e *EmailService) SendOrderConfirmationEmail(to, orderID string, tickets []string, totalPrice float64) error {
	subject := fmt.Sprintf("Order Confirmation - %s", orderID)

	ticketList := ""
	for _, ticket := range tickets {
		ticketList += fmt.Sprintf("<div class=\"ticket-item\">%s</div>", ticket)
	}

	body := fmt.Sprintf(`
        <h3>Thank you for your order!</h3>
        <p><strong>Order ID:</strong> %s</p>
        <p><strong>Total Amount:</strong> $%.2f</p>
        
        <h4>Your Tickets:</h4>
        %s
        
        <p>Your tickets have been confirmed. Please keep this email for your records.</p>
        <p>We look forward to seeing you at the event!</p>
    `, orderID, totalPrice, ticketList)

	return e.SendEmail(to, subject, body)
}
