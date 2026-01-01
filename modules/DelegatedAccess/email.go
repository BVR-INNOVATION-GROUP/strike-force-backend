package delegatedaccess

import (
	"fmt"
	"os"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// SendDelegatedAccessEmail sends an email with login credentials to a delegated user
func SendDelegatedAccessEmail(email, name, password, organizationName, delegatorName string) error {
	mailjetKey := os.Getenv("MAILJET_KEY")
	mailjetSecret := os.Getenv("MAILJET_SECRET")
	mailjetEmail := os.Getenv("MAILJET_EMAIL")
	mailjetFrom := os.Getenv("MAILJET_FROM")

	if mailjetKey == "" || mailjetSecret == "" || mailjetEmail == "" {
		return fmt.Errorf("mailjet configuration is missing")
	}

	if mailjetFrom == "" {
		mailjetFrom = "StrikeForce"
	}

	mj := mailjet.NewMailjetClient(mailjetKey, mailjetSecret)

	// Get frontend URL from centralized config
	baseURL := core.GetFrontendURL()
	loginURL := fmt.Sprintf("%s/auth/login", baseURL)

	subject := fmt.Sprintf("Delegated Access to %s on StrikeForce", organizationName)
	textPart := fmt.Sprintf(
		"Hello %s,\n\n"+
			"%s has granted you delegated access to manage %s on StrikeForce.\n\n"+
			"Your login credentials:\n"+
			"Email: %s\n"+
			"Password: %s\n\n"+
			"Login here: %s\n\n"+
			"You will have access to the same university admin dashboard and can manage the organization.\n\n"+
			"Please log in and change your password as soon as possible.\n\n"+
			"Best regards,\n"+
			"The StrikeForce Team",
		name, delegatorName, organizationName, email, password, loginURL,
	)

	htmlPart := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Delegated Access Granted</h2>
			<p>Hello %s,</p>
			<p><strong>%s</strong> has granted you delegated access to manage <strong>%s</strong> on StrikeForce.</p>
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
			<p>You will have access to the same university admin dashboard and can manage the organization.</p>
			<p><em>Please log in and change your password as soon as possible.</em></p>
			<p>Best regards,<br>The StrikeForce Team</p>
		</div>`,
		name, delegatorName, organizationName, email, password, loginURL, loginURL,
	)

	message := mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: mailjetEmail,
			Name:  mailjetFrom,
		},
		To: &mailjet.RecipientsV31{
			{
				Email: email,
				Name:  name,
			},
		},
		Subject:  subject,
		TextPart: textPart,
		HTMLPart: htmlPart,
	}

	messages := mailjet.MessagesV31{
		Info: []mailjet.InfoMessagesV31{message},
	}

	_, err := mj.SendMailV31(&messages)
	return mailer.InterpretMailjetError(err, "delegated access email")
}


