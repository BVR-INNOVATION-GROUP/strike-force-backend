package admin

import (
	"fmt"
	"os"
	"strings"
	"time"

	core "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Core"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	mailer "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Mailer"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"github.com/mailjet/mailjet-apiv3-go/v4"
	"gorm.io/gorm"
)

// SuperAdminDelegation records that a super-admin invited another super-admin (platform scope, no organization).
type SuperAdminDelegation struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	DelegatedUserID uint           `json:"delegatedUserId" gorm:"not null;index;uniqueIndex:idx_super_admin_delegation_pair"`
	DelegatorID     uint           `json:"delegatorId" gorm:"not null;index;uniqueIndex:idx_super_admin_delegation_pair"`
	IsActive        bool           `json:"isActive" gorm:"default:true"`
	DelegatedUser   user.User      `json:"delegatedUser" gorm:"foreignKey:DelegatedUserID"`
	Delegator       user.User      `json:"delegator" gorm:"foreignKey:DelegatorID"`
}

func (SuperAdminDelegation) TableName() string { return "super_admin_delegations" }

// Slim JSON for API — avoids encoding full user.User (embedded profile, groups, etc.).
type delegationUserOut struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
}

type superAdminDelegationOut struct {
	ID              uint              `json:"id"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	DelegatedUserID uint              `json:"delegatedUserId"`
	DelegatorID     uint              `json:"delegatorId"`
	IsActive        bool              `json:"isActive"`
	DelegatedUser   delegationUserOut `json:"delegatedUser"`
	Delegator       delegationUserOut `json:"delegator"`
}

func toSuperAdminDelegationOut(row SuperAdminDelegation) superAdminDelegationOut {
	return superAdminDelegationOut{
		ID:              row.ID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DelegatedUserID: row.DelegatedUserID,
		DelegatorID:     row.DelegatorID,
		IsActive:        row.IsActive,
		DelegatedUser: delegationUserOut{
			ID:    row.DelegatedUser.ID,
			Email: row.DelegatedUser.Email,
			Name:  row.DelegatedUser.Name,
			Role:  row.DelegatedUser.Role,
		},
		Delegator: delegationUserOut{
			ID:    row.Delegator.ID,
			Email: row.Delegator.Email,
			Name:  row.Delegator.Name,
			Role:  row.Delegator.Role,
		},
	}
}

type createSuperAdminDelegationReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// sendSuperAdminDelegationEmail emails credentials for a newly created super-admin delegate.
func sendSuperAdminDelegationEmail(email, name, password, delegatorName string) error {
	if mailer.IsDevMode() {
		fmt.Printf("Dev mode: super-admin delegation email for %s (password: %s)\n", email, password)
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

	baseURL := core.GetFrontendURL()
	loginURL := fmt.Sprintf("%s/auth/login", baseURL)
	if delegatorName == "" {
		delegatorName = "a StrikeForce administrator"
	}

	subject := "Super admin access to StrikeForce"
	textPart := fmt.Sprintf(
		"Hello %s,\n\n"+
			"%s has granted you super administrator access to StrikeForce.\n\n"+
			"Your login credentials:\nEmail: %s\nPassword: %s\n\nLogin: %s\n\n"+
			"Please sign in and change your password as soon as possible.\n\n"+
			"The StrikeForce Team",
		name, delegatorName, email, password, loginURL,
	)
	htmlPart := fmt.Sprintf(
		`<div style="font-family: Arial, sans-serif; max-width: 600px;">
			<h2>Super administrator access</h2>
			<p>Hello %s,</p>
			<p><strong>%s</strong> has granted you <strong>super administrator</strong> access to StrikeForce.</p>
			<p><strong>Email:</strong> %s<br><strong>Password:</strong> <code>%s</code></p>
			<p><a href="%s" style="background:#e9226e;color:#fff;padding:12px 24px;text-decoration:none;border-radius:4px;display:inline-block;">Sign in</a></p>
			<p style="font-size:12px;color:#666;">%s</p>
			<p><em>Please change your password after signing in.</em></p>
		</div>`,
		name, delegatorName, email, password, loginURL, loginURL,
	)

	mj := mailjet.NewMailjetClient(mailjetKey, mailjetSecret)
	message := mailjet.InfoMessagesV31{
		From:     &mailjet.RecipientV31{Email: mailjetEmail, Name: mailjetFrom},
		To:       &mailjet.RecipientsV31{{Email: email, Name: name}},
		Subject:  subject,
		TextPart: textPart,
		HTMLPart: htmlPart,
	}
	_, err := mj.SendMailV31(&mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{message}})
	return mailer.InterpretMailjetError(err, "super-admin delegation email")
}

// CreateSuperAdminDelegation invites or links another super-admin (delegator = current user).
func CreateSuperAdminDelegation(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	var req createSuperAdminDelegationReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request: " + err.Error()})
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "email and name are required"})
	}

	var delegator user.User
	if err := db.First(&delegator, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "delegator not found"})
	}

	var existing user.User
	if err := db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		if existing.ID == userID {
			return c.Status(400).JSON(fiber.Map{"msg": "you cannot delegate access to yourself"})
		}
		if existing.Role != "super-admin" {
			return c.Status(400).JSON(fiber.Map{"msg": "this email belongs to a user who is not a super administrator"})
		}
		var dup SuperAdminDelegation
		if err := db.Where("delegator_id = ? AND delegated_user_id = ?", userID, existing.ID).First(&dup).Error; err == nil {
			return c.Status(400).JSON(fiber.Map{"msg": "you have already delegated this super administrator"})
		}
		row := SuperAdminDelegation{
			DelegatedUserID: existing.ID,
			DelegatorID:     userID,
			IsActive:        true,
		}
		if err := db.Create(&row).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to record delegation: " + err.Error()})
		}
		db.Preload("DelegatedUser").Preload("Delegator").First(&row, row.ID)
		return c.Status(201).JSON(fiber.Map{
			"msg":  "super-admin delegation recorded (existing user)",
			"data": toSuperAdminDelegationOut(row),
		})
	}

	pw, err := delegatedaccess.GenerateRandomPassword(12)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to generate password"})
	}
	newUser := user.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: user.GenerateHash(pw),
		Role:     "super-admin",
	}
	if err := db.Create(&newUser).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create user: " + err.Error()})
	}
	row := SuperAdminDelegation{
		DelegatedUserID: newUser.ID,
		DelegatorID:     userID,
		IsActive:        true,
	}
	if err := db.Create(&row).Error; err != nil {
		db.Unscoped().Delete(&newUser)
		return c.Status(400).JSON(fiber.Map{"msg": "failed to record delegation: " + err.Error()})
	}
	db.Preload("DelegatedUser").Preload("Delegator").First(&row, row.ID)

	dname := delegator.Name
	if dname == "" {
		dname = delegator.Email
	}
	if err := sendSuperAdminDelegationEmail(newUser.Email, newUser.Name, pw, dname); err != nil {
		fmt.Printf("super-admin delegation email failed: %v\n", err)
	}

	return c.Status(201).JSON(fiber.Map{
		"msg":  "super-admin created and invitation sent",
		"data": toSuperAdminDelegationOut(row),
	})
}

// ListSuperAdminDelegations returns delegations created by the current super-admin.
func ListSuperAdminDelegations(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	var rows []SuperAdminDelegation
	if err := db.Where("delegator_id = ?", userID).
		Preload("DelegatedUser").Preload("Delegator").
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to list delegations: " + err.Error()})
	}
	out := make([]superAdminDelegationOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSuperAdminDelegationOut(row))
	}
	return c.JSON(fiber.Map{"data": out})
}

// DeleteSuperAdminDelegation removes a delegation record created by the current user.
func DeleteSuperAdminDelegation(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "id required"})
	}
	var row SuperAdminDelegation
	if err := db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "delegation not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": err.Error()})
	}
	if row.DelegatorID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you can only remove delegations you created"})
	}
	if err := db.Delete(&row).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to remove delegation: " + err.Error()})
	}
	return c.JSON(fiber.Map{"msg": "delegation removed"})
}
