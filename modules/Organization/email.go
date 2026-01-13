package organization

import (
	"fmt"
	"os"
	"strings"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	"github.com/mailjet/mailjet-apiv3-go/v4"
)

func statusSubject(status string) string {
	switch status {
	case "APPROVED":
		return "Your organization has been approved"
	case "REJECTED":
		return "Your organization review outcome"
	default:
		return "Your organization status was updated"
	}
}

func SendOrganizationStatusEmail(org Organization, recipient string, previousApproved bool) error {
	if recipient == "" {
		return nil
	}

	status := normalizeKYCStatus(org.IsApproved)
	prevStatus := normalizeKYCStatus(previousApproved)
	subject := statusSubject(status)

	body := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>%s</h2>
			<p>Hello %s,</p>
			<p>This is to notify you that the verification status of <strong>%s</strong> has been updated.</p>
			<p><strong>New status:</strong> %s</p>
			%s
			<p>If you have any questions, please contact the StrikeForce support team.</p>
		</div>`,
		subject,
		org.Name,
		org.Name,
		status,
		buildStatusHint(status, prevStatus),
	)

	text := fmt.Sprintf(
		"Hello %s,\n\nThe verification status of %s has been updated to %s.\n\nIf you have any questions, contact the StrikeForce support team.",
		org.Name, org.Name, status,
	)

	return sendOrganizationEmail(recipient, subject, body, text, "org status")
}

func buildStatusHint(newStatus, previousStatus string) string {
	switch newStatus {
	case "APPROVED":
		return "<p>You now have full access to the platform. Welcome aboard!</p>"
	case "REJECTED":
		return "<p>Please review your submission and contact support if you believe this is an error.</p>"
	default:
		if strings.ToUpper(previousStatus) == "APPROVED" {
			return "<p>Your organization has been moved back to pending review. Our team will contact you with next steps.</p>"
		}
		return ""
	}
}

func SendOrganizationUpdateEmail(org Organization, recipient string) error {
	if recipient == "" {
		return nil
	}

	subject := "Your organization profile was updated"
	body := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>%s</h2>
			<p>Hello %s,</p>
			<p>We wanted to let you know that the profile for <strong>%s</strong> was updated.</p>
			<p>If you did not request this change, please contact the StrikeForce support team immediately.</p>
		</div>`,
		subject,
		org.Name,
		org.Name,
	)

	text := fmt.Sprintf(
		"Hello %s,\n\nThe profile for %s was updated. If you did not request this change please contact StrikeForce support.",
		org.Name,
		org.Name,
	)

	return sendOrganizationEmail(recipient, subject, body, text, "org update")
}

func sendOrganizationEmail(to, subject, html, text, context string) error {
	// Skip sending email in dev mode
	if mailer.IsDevMode() {
		fmt.Printf("Dev mode: Skipping organization email (%s) to %s\n", context, to)
		return nil
	}

	mailjetKey := os.Getenv("MAILJET_KEY")
	mailjetSecret := os.Getenv("MAILJET_SECRET")
	mailjetEmail := os.Getenv("MAILJET_EMAIL")
	mailjetFrom := os.Getenv("MAILJET_FROM")

	if mailjetKey == "" || mailjetSecret == "" || mailjetEmail == "" {
		return fmt.Errorf("mailjet configuration missing")
	}

	if mailjetFrom == "" {
		mailjetFrom = "StrikeForce"
	}

	client := mailjet.NewMailjetClient(mailjetKey, mailjetSecret)

	// Extract name from recipient if it's in format "Name <email>" or use email as name
	recipientName := to
	if strings.Contains(to, "<") {
		parts := strings.Split(to, "<")
		if len(parts) == 2 {
			recipientName = strings.TrimSpace(parts[0])
			to = strings.Trim(strings.TrimSpace(parts[1]), ">")
		}
	}

	message := mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: mailjetEmail,
			Name:  mailjetFrom,
		},
		To: &mailjet.RecipientsV31{
			{
				Email: to,
				Name:  recipientName,
			},
		},
		Subject:  subject,
		HTMLPart: html,
		TextPart: text,
	}

	msgs := mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{message}}
	_, err := client.SendMailV31(&msgs)
	return mailer.InterpretMailjetError(err, context)
}

// SendOrganizationCreationEmail sends a welcome email to the organization owner with login credentials
func SendOrganizationCreationEmail(org Organization, ownerEmail, ownerName, password string) error {
	if ownerEmail == "" {
		return nil
	}

	// Get frontend URL from centralized config
	baseURL := core.GetFrontendURL()
	loginURL := fmt.Sprintf("%s/auth/login", baseURL)

	orgTypeLabel := "Organization"
	if strings.ToLower(org.Type) == "university" {
		orgTypeLabel = "University"
	} else if strings.ToLower(org.Type) == "company" {
		orgTypeLabel = "Partner Organization"
	}

	subject := fmt.Sprintf("Welcome to StrikeForce - Your %s Account", orgTypeLabel)

	displayName := strings.TrimSpace(ownerName)
	if displayName == "" {
		// Extract name from email if no name provided
		emailParts := strings.Split(ownerEmail, "@")
		if len(emailParts) > 0 {
			displayName = emailParts[0]
		} else {
			displayName = ownerEmail
		}
	}

	body := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Welcome to StrikeForce!</h2>
			<p>Hello %s,</p>
			<p>Your %s account for <strong>%s</strong> has been created on StrikeForce.</p>
			<p>Your organization has been pre-approved and is ready to use.</p>
			<h3>Your login credentials:</h3>
			<p><strong>Email:</strong> %s<br>
			<strong>Password:</strong> <code>%s</code></p>
			<p style="margin: 30px 0;">
				<a href="%s" style="background-color: #e9226e; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">
					Login to StrikeForce
				</a>
			</p>
			<p>Or copy and paste this link into your browser:</p>
			<p style="color: #666; font-size: 12px; word-break: break-all;">%s</p>
			<p><em>Please log in and change your password as soon as possible.</em></p>
			<p>If you have any questions, please contact the StrikeForce support team.</p>
			<p>Best regards,<br>The StrikeForce Team</p>
		</div>`,
		displayName,
		orgTypeLabel,
		org.Name,
		ownerEmail,
		password,
		loginURL,
		loginURL,
	)

	text := fmt.Sprintf(
		"Hello %s,\n\n"+
			"Your %s account for %s has been created on StrikeForce.\n\n"+
			"Your organization has been pre-approved and is ready to use.\n\n"+
			"Your login credentials:\n"+
			"Email: %s\n"+
			"Password: %s\n\n"+
			"Login here: %s\n\n"+
			"Please log in and change your password as soon as possible.\n\n"+
			"If you have any questions, please contact the StrikeForce support team.\n\n"+
			"Best regards,\n"+
			"The StrikeForce Team",
		displayName,
		orgTypeLabel,
		org.Name,
		ownerEmail,
		password,
		loginURL,
	)

	return sendOrganizationEmail(ownerEmail, subject, body, text, "org creation")
}
