package builders

import (
	"fmt"
	"strings"
)

// EmailBuilder provides methods to build HTML email templates
type EmailBuilder struct {
	styles     string
	header     string
	content    []string
	footer     string
	brandName  string
	brandColor string
}

// NewEmailBuilder creates a new email builder with default styling
func NewEmailBuilder(brandName, brandColor string) *EmailBuilder {
	if brandName == "" {
		brandName = "Ticketly"
	}
	if brandColor == "" {
		brandColor = "#4F46E5" // Default indigo color
	}

	return &EmailBuilder{
		brandName:  brandName,
		brandColor: brandColor,
		content:    make([]string, 0),
	}
}

// SetStyles sets the CSS styles for the email
func (b *EmailBuilder) SetStyles(styles string) *EmailBuilder {
	b.styles = styles
	return b
}

// SetHeader sets the email header
func (b *EmailBuilder) SetHeader(title string, subtitle string) *EmailBuilder {
	b.header = fmt.Sprintf(`
		<div class="header">
			<h1 style="color: %s; margin: 0; font-size: 28px;">%s</h1>
			%s
		</div>
	`, b.brandColor, title, b.conditionalSubtitle(subtitle))
	return b
}

// AddSection adds a content section to the email
func (b *EmailBuilder) AddSection(title string, content string) *EmailBuilder {
	section := fmt.Sprintf(`
		<div class="section">
			<h2 style="color: %s; font-size: 20px; margin-bottom: 15px;">%s</h2>
			<div class="section-content">%s</div>
		</div>
	`, b.brandColor, title, content)
	b.content = append(b.content, section)
	return b
}

// AddInfoBox adds an information box (for important details)
func (b *EmailBuilder) AddInfoBox(content string, boxType string) *EmailBuilder {
	var bgColor, borderColor string
	switch boxType {
	case "success":
		bgColor = "#DEF7EC"
		borderColor = "#03543F"
	case "warning":
		bgColor = "#FEF3C7"
		borderColor = "#92400E"
	case "error":
		bgColor = "#FEE2E2"
		borderColor = "#991B1B"
	case "info":
		bgColor = "#DBEAFE"
		borderColor = "#1E40AF"
	default:
		bgColor = "#F3F4F6"
		borderColor = "#6B7280"
	}

	infoBox := fmt.Sprintf(`
		<div style="background-color: %s; border-left: 4px solid %s; padding: 15px; margin: 20px 0; border-radius: 4px;">
			%s
		</div>
	`, bgColor, borderColor, content)
	b.content = append(b.content, infoBox)
	return b
}

// AddDetailsList adds a list of key-value details
func (b *EmailBuilder) AddDetailsList(details map[string]string) *EmailBuilder {
	var items []string
	for key, value := range details {
		items = append(items, fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 12px; font-weight: bold; color: #4B5563;">%s:</td>
				<td style="padding: 8px 12px; color: #1F2937;">%s</td>
			</tr>
		`, key, value))
	}

	detailsList := fmt.Sprintf(`
		<table style="width: 100%%; border-collapse: collapse; margin: 15px 0;">
			%s
		</table>
	`, strings.Join(items, ""))
	b.content = append(b.content, detailsList)
	return b
}

// AddButton adds a call-to-action button
func (b *EmailBuilder) AddButton(text, url string) *EmailBuilder {
	button := fmt.Sprintf(`
		<div style="text-align: center; margin: 30px 0;">
			<a href="%s" style="display: inline-block; background-color: %s; color: white; 
				padding: 12px 30px; text-decoration: none; border-radius: 6px; font-weight: bold;">
				%s
			</a>
		</div>
	`, url, b.brandColor, text)
	b.content = append(b.content, button)
	return b
}

// AddDivider adds a horizontal divider
func (b *EmailBuilder) AddDivider() *EmailBuilder {
	divider := `<hr style="border: none; border-top: 1px solid #E5E7EB; margin: 30px 0;">`
	b.content = append(b.content, divider)
	return b
}

// AddParagraph adds a simple paragraph
func (b *EmailBuilder) AddParagraph(text string) *EmailBuilder {
	paragraph := fmt.Sprintf(`<p style="line-height: 1.6; color: #4B5563; margin: 15px 0;">%s</p>`, text)
	b.content = append(b.content, paragraph)
	return b
}

// SetFooter sets the email footer
func (b *EmailBuilder) SetFooter(footerText string) *EmailBuilder {
	if footerText == "" {
		footerText = fmt.Sprintf(`
			<p>Thank you for using %s!</p>
			<p style="font-size: 11px; color: #9CA3AF; margin-top: 10px;">
				This is an automated email. Please do not reply to this message.
			</p>
		`, b.brandName)
	}
	b.footer = fmt.Sprintf(`<div class="footer">%s</div>`, footerText)
	return b
}

// Build constructs the final HTML email
func (b *EmailBuilder) Build() string {
	if b.styles == "" {
		b.styles = b.getDefaultStyles()
	}
	if b.footer == "" {
		b.SetFooter("")
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Email</title>
	<style>%s</style>
</head>
<body>
	<div class="container">
		%s
		<div class="content">
			%s
		</div>
		%s
	</div>
</body>
</html>
	`, b.styles, b.header, strings.Join(b.content, "\n"), b.footer)
}

// getDefaultStyles returns default CSS styles
func (b *EmailBuilder) getDefaultStyles() string {
	return `
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
			line-height: 1.6;
			color: #1F2937;
			background-color: #F9FAFB;
			margin: 0;
			padding: 0;
		}
		.container {
			max-width: 600px;
			margin: 20px auto;
			background-color: #FFFFFF;
			border-radius: 8px;
			box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
			overflow: hidden;
		}
		.header {
			background: linear-gradient(135deg, ` + b.brandColor + ` 0%, ` + b.darkenColor(b.brandColor) + ` 100%);
			color: white;
			padding: 30px 20px;
			text-align: center;
		}
		.content {
			padding: 30px 20px;
		}
		.footer {
			text-align: center;
			padding: 20px;
			border-top: 1px solid #E5E7EB;
			color: #6B7280;
			font-size: 14px;
			background-color: #F9FAFB;
		}
		.section {
			margin-bottom: 25px;
		}
		.section-content {
			color: #4B5563;
		}
		h1, h2, h3 {
			margin: 0 0 10px 0;
		}
		a {
			color: ` + b.brandColor + `;
			text-decoration: none;
		}
		a:hover {
			text-decoration: underline;
		}
	`
}

// conditionalSubtitle returns subtitle HTML if subtitle is not empty
func (b *EmailBuilder) conditionalSubtitle(subtitle string) string {
	if subtitle == "" {
		return ""
	}
	return fmt.Sprintf(`<p style="margin: 10px 0 0 0; font-size: 16px; opacity: 0.95;">%s</p>`, subtitle)
}

// darkenColor darkens a hex color (simple implementation)
func (b *EmailBuilder) darkenColor(color string) string {
	// Simple darkening - in production, use proper color manipulation
	colorMap := map[string]string{
		"#4F46E5": "#4338CA",
		"#3B82F6": "#2563EB",
		"#10B981": "#059669",
		"#F59E0B": "#D97706",
		"#EF4444": "#DC2626",
	}
	if darker, ok := colorMap[color]; ok {
		return darker
	}
	return color
}
