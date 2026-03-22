package directmessage

import (
	"errors"
	"strconv"
	"strings"
	"time"

	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// isDuplicateKeyError matches PostgreSQL unique violations (concurrent create or soft-deleted row still indexed).
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "23505") || strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

func orderedPair(a, b uint) (uint, uint) {
	if a < b {
		return a, b
	}
	return b, a
}

func isThreadParticipant(thread *Thread, uid uint) bool {
	return thread.UserASmall == uid || thread.UserBLarge == uid
}

// CreateOrGetThread — super-admin only; target must be partner or university-admin.
func CreateOrGetThread(c *fiber.Ctx, db *gorm.DB) error {
	role, _ := c.Locals("role").(string)
	if role != "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "only super-admin can start admin direct messages"})
	}
	callerID := c.Locals("user_id").(uint)

	type reqBody struct {
		TargetUserID uint `json:"targetUserId"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil || req.TargetUserID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "targetUserId is required"})
	}
	if req.TargetUserID == callerID {
		return c.Status(400).JSON(fiber.Map{"msg": "cannot message yourself"})
	}

	var target user.User
	if err := db.First(&target, req.TargetUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load user"})
	}
	if target.Role != "partner" && target.Role != "university-admin" {
		return c.Status(400).JSON(fiber.Map{"msg": "you can only message partner or university-admin accounts"})
	}

	small, large := orderedPair(callerID, req.TargetUserID)
	var thread Thread
	err := db.Where("user_a_small = ? AND user_b_large = ?", small, large).First(&thread).Error
	if err == nil {
		// existing active thread
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		thread = Thread{
			UserASmall:  small,
			UserBLarge:  large,
			CreatedByID: callerID,
		}
		if err := db.Create(&thread).Error; err != nil {
			if !isDuplicateKeyError(err) {
				return c.Status(400).JSON(fiber.Map{"msg": "failed to create thread: " + err.Error()})
			}
			// Another request created the row, or a soft-deleted row still holds the unique pair
			if err2 := db.Unscoped().Where("user_a_small = ? AND user_b_large = ?", small, large).First(&thread).Error; err2 != nil {
				return c.Status(400).JSON(fiber.Map{"msg": "failed to resolve thread: " + err2.Error()})
			}
			if thread.DeletedAt.Valid {
				if err3 := db.Unscoped().Model(&thread).Update("deleted_at", nil).Error; err3 != nil {
					return c.Status(400).JSON(fiber.Map{"msg": "failed to restore thread: " + err3.Error()})
				}
			}
		}
		if err := db.First(&thread, thread.ID).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to load thread: " + err.Error()})
		}
	} else {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to resolve thread: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"id":           thread.ID,
		"targetUserId": req.TargetUserID,
	}})
}

// ListThreads for current user (participant).
func ListThreads(c *fiber.Ctx, db *gorm.DB) error {
	uid := c.Locals("user_id").(uint)

	var threads []Thread
	if err := db.Where("user_a_small = ? OR user_b_large = ?", uid, uid).
		Order("updated_at DESC").
		Find(&threads).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to list threads: " + err.Error()})
	}

	out := make([]fiber.Map, 0, len(threads))
	for _, t := range threads {
		otherID := t.UserBLarge
		if t.UserBLarge == uid {
			otherID = t.UserASmall
		}
		var other user.User
		_ = db.First(&other, otherID).Error
		var last Message
		_ = db.Where("thread_id = ?", t.ID).Order("created_at DESC").First(&last).Error
		lastSenderID := uint(0)
		lastAtStr := ""
		lastFromOther := false
		if last.ID != 0 {
			lastSenderID = last.SenderID
			lastAtStr = last.CreatedAt.Format(time.RFC3339Nano)
			lastFromOther = last.SenderID != uid
		}
		out = append(out, fiber.Map{
			"id":              t.ID,
			"otherUserId":     otherID,
			"otherName":       other.Name,
			"otherEmail":      other.Email,
			"otherRole":       other.Role,
			"lastBody":        last.Body,
			"lastAt":          lastAtStr,
			"lastSenderId":    lastSenderID,
			"lastFromOther":   lastFromOther,
			"updatedAt":       t.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	return c.JSON(fiber.Map{"data": out})
}

// ListMessages returns messages for a thread (participant only).
func ListMessages(c *fiber.Ctx, db *gorm.DB) error {
	uid := c.Locals("user_id").(uint)
	id := c.Params("id")
	var thread Thread
	if err := db.First(&thread, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "thread not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load thread"})
	}
	if !isThreadParticipant(&thread, uid) {
		return c.Status(403).JSON(fiber.Map{"msg": "forbidden"})
	}

	var messages []Message
	if err := db.Where("thread_id = ?", thread.ID).Preload("Sender").Order("created_at ASC").Find(&messages).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load messages: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": messages})
}

// SendMessage posts a message (participant only).
func SendMessage(c *fiber.Ctx, db *gorm.DB) error {
	uid := c.Locals("user_id").(uint)
	id := c.Params("id")

	var thread Thread
	if err := db.First(&thread, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "thread not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load thread"})
	}
	if !isThreadParticipant(&thread, uid) {
		return c.Status(403).JSON(fiber.Map{"msg": "forbidden"})
	}

	type reqBody struct {
		Body string `json:"body"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid body"})
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "message body is required"})
	}

	msg := Message{
		ThreadID: thread.ID,
		SenderID: uid,
		Body:     req.Body,
	}
	if err := db.Create(&msg).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to send: " + err.Error()})
	}
	_ = db.Model(&thread).Update("updated_at", time.Now()).Error
	db.Preload("Sender").First(&msg, msg.ID)

	return c.Status(201).JSON(fiber.Map{"data": msg})
}

// ResolveUniversityAdminUser returns the org owner user id for a university organization.
func ResolveUniversityAdminUser(db *gorm.DB, universityOrgID uint) (uint, error) {
	var org organization.Organization
	if err := db.First(&org, universityOrgID).Error; err != nil {
		return 0, err
	}
	if org.UserID == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return org.UserID, nil
}

// GetUniversityAdminForOrg returns the primary admin user for an organization (super-admin only).
func GetUniversityAdminForOrg(c *fiber.Ctx, db *gorm.DB) error {
	role, _ := c.Locals("role").(string)
	if role != "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "forbidden"})
	}
	orgIDStr := c.Params("orgId")
	orgID64, err := strconv.ParseUint(orgIDStr, 10, 32)
	if err != nil || orgID64 == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid organization id"})
	}
	uid, err := ResolveUniversityAdminUser(db, uint(orgID64))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "organization or admin not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": err.Error()})
	}
	var u user.User
	if err := db.First(&u, uid).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
	}
	return c.JSON(fiber.Map{"data": fiber.Map{
		"userId": u.ID,
		"name":   u.Name,
		"email":  u.Email,
		"role":   u.Role,
	}})
}
