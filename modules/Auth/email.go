package auth

import (
	"fmt"
	"os"
	"strings"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// SendPasswordResetEmail dispatches the reset link to the user.
func SendPasswordResetEmail(email, token, name string) error {
	// Skip sending email in dev mode
	if mailer.IsDevMode() {
		fmt.Printf("Dev mode: Skipping password reset email for %s (token: %s)\n", email, token)
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

	// Get frontend URL from centralized config
	baseURL := core.GetFrontendURL()
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, token)

	displayName := strings.TrimSpace(name)
	if displayName != "" {
		displayName = " " + displayName
	}

	subject := "Reset your StrikeForce password"
	htmlPart := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Reset Your Password</h2>
			<p>Hello%s,</p>
			<p>We received a request to reset your password for your StrikeForce account.</p>
			<p>Click the link below to reset your password:</p>
			<p style="margin: 30px 0;">
				<a href="%s" style="background-color: #e9226e; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">
					Reset Password
				</a>
			</p>
			<p>Or copy and paste this link into your browser:</p>
			<p style="color: #666; font-size: 12px; word-break: break-all;">%s</p>
			<p style="margin-top: 30px; color: #666; font-size: 12px;">
				This link will expire in 1 hour. If you didn't request a reset, please ignore this email.
			</p>
		</div>`,
		displayName,
		resetURL,
		resetURL,
	)

	textPart := fmt.Sprintf(
		"Hello%s,\n\nWe received a request to reset your password. Reset it here: %s\n\nThis link expires in 1 hour.",
		displayName,
		resetURL,
	)

	mj := mailjet.NewMailjetClient(mailjetKey, mailjetSecret)
	message := mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: mailjetEmail,
			Name:  mailjetFrom,
		},
		To: &mailjet.RecipientsV31{
			{
				Email: email,
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
	return mailer.InterpretMailjetError(err, "password reset email")
}
