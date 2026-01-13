package invitation

import (
	"fmt"
	"os"
	"strings"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// SendInvitationEmail sends an invitation email with the invitation link
func SendInvitationEmail(email, token, name, role, organizationName string) error {
	// Skip sending email in dev mode
	if mailer.IsDevMode() {
		fmt.Printf("Dev mode: Skipping invitation email for %s (token: %s)\n", email, token)
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
	inviteURL := fmt.Sprintf("%s/auth/invite?token=%s", baseURL, token)

	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = email
	} else {
		displayName = " " + displayName
	}

	roleLabel := role
	if roleLabel == "" {
		roleLabel = "User"
	} else {
		// Capitalize first letter
		if len(roleLabel) > 0 {
			roleLabel = strings.ToUpper(roleLabel[:1]) + strings.ToLower(roleLabel[1:])
		}
	}

	subject := fmt.Sprintf("You've been invited to join %s on StrikeForce", organizationName)
	htmlPart := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>You've Been Invited!</h2>
			<p>Hello%s,</p>
			<p>You have been invited to join <strong>%s</strong> as a %s on StrikeForce.</p>
			<p>Click the link below to accept your invitation and create your account:</p>
			<p style="margin: 30px 0;">
				<a href="%s" style="background-color: #e9226e; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">
					Accept Invitation
				</a>
			</p>
			<p>Or copy and paste this link into your browser:</p>
			<p style="color: #666; font-size: 12px; word-break: break-all;">%s</p>
			<p style="margin-top: 30px; color: #666; font-size: 12px;">
				This invitation will expire in 7 days. If you didn't expect this invitation, please ignore this email.
			</p>
		</div>`,
		displayName,
		organizationName,
		roleLabel,
		inviteURL,
		inviteURL,
	)

	textPart := fmt.Sprintf(
		"Hello%s,\n\nYou have been invited to join %s as a %s on StrikeForce.\n\nAccept your invitation here: %s\n\nThis invitation expires in 7 days.",
		displayName,
		organizationName,
		roleLabel,
		inviteURL,
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
	return mailer.InterpretMailjetError(err, "invitation email")
}
