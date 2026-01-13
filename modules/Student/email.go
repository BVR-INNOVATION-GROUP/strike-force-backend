package student

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// GenerateRandomPassword generates a secure random password
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 12 // Default to 12 characters
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Use base64 encoding for a readable password
	password := base64.URLEncoding.EncodeToString(bytes)
	// Take only the first 'length' characters and ensure it's alphanumeric
	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

// SendPasswordEmail sends an email with the password to the student
func SendPasswordEmail(studentEmail, studentName, password string) error {
	// Skip sending email in dev mode
	if mailer.IsDevMode() {
		fmt.Printf("Dev mode: Skipping password email for %s (password: %s)\n", studentEmail, password)
		return nil
	}

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

	subject := "Welcome to StrikeForce - Your Account Credentials"
	textPart := fmt.Sprintf(
		"Hello %s,\n\n"+
			"Your student account has been created on StrikeForce.\n\n"+
			"Your login credentials:\n"+
			"Email: %s\n"+
			"Password: %s\n\n"+
			"Login here: %s\n\n"+
			"Please log in and change your password as soon as possible.\n\n"+
			"Best regards,\n"+
			"The StrikeForce Team",
		studentName, studentEmail, password, loginURL,
	)

	htmlPart := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Welcome to StrikeForce!</h2>
			<p>Hello %s,</p>
			<p>Your student account has been created on StrikeForce.</p>
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
			<p>Best regards,<br>The StrikeForce Team</p>
		</div>`,
		studentName, studentEmail, password, loginURL, loginURL,
	)

	message := mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: mailjetEmail,
			Name:  mailjetFrom,
		},
		To: &mailjet.RecipientsV31{
			{
				Email: studentEmail,
				Name:  studentName,
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
	return mailer.InterpretMailjetError(err, "student password email")
}
