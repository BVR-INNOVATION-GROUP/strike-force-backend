# Backend Implementation Guide

This guide outlines the missing backend endpoints that need to be implemented to fully support the frontend.

## Overview

The frontend has been updated to connect directly to the Go backend API. All repositories now call backend endpoints at `/api/v1/*` or `/user/*`. This document lists the endpoints that need to be implemented.

## Response Format

All backend endpoints should return data in this format:
```go
c.JSON(fiber.Map{
    "msg": "success message",
    "data": <actual data>,
})
```

For errors:
```go
c.Status(400).JSON(fiber.Map{
    "msg": "error message",
})
```

## Missing Endpoints by Module

### 1. Application Module (`/api/v1/applications`)

**Status**: Module exists but router is empty. Needs full implementation.

**Required Endpoints:**
- `GET /api/v1/applications` - Get all applications (with optional query params: `projectId`, `userId`)
- `GET /api/v1/applications/:id` - Get application by ID
- `POST /api/v1/applications` - Create new application
- `PUT /api/v1/applications/:id` - Update application
- `DELETE /api/v1/applications/:id` - Delete application
- `POST /api/v1/applications/upload` - Upload application files (FormData with `files` field)

**Model Fields Needed:**
- ProjectID
- StudentIDs (array)
- Status (PENDING, REVIEWED, OFFERED, ACCEPTED, DECLINED)
- PortfolioScore
- Attachments (file paths)
- AppliedAt
- OfferExpiresAt (optional)

### 2. Invitation Module (`/api/v1/invitations`)

**Status**: Module exists but router is empty. Needs full implementation.

**Required Endpoints:**
- `GET /api/v1/invitations` - Get all invitations (with optional query param: `universityId`)
- `GET /api/v1/invitations/:id` - Get invitation by ID
- `GET /api/v1/invitations/token/:token` - Get invitation by token (public endpoint, no auth)
- `POST /api/v1/invitations` - Create new invitation
- `POST /api/v1/invitations/accept` - Accept invitation and create user (public endpoint)
- `PUT /api/v1/invitations/:id` - Update invitation
- `DELETE /api/v1/invitations/:id` - Delete invitation

**Model Fields Needed:**
- Email
- Role (student, supervisor)
- UniversityID
- Token (unique, secure)
- Status (PENDING, USED, EXPIRED)
- ExpiresAt
- UsedAt (optional)

### 3. Project Module (`/api/v1/projects`)

**Status**: Partially implemented. Missing some endpoints.

**Existing Endpoints:**
- ✅ `POST /api/v1/projects` - Create project
- ✅ `GET /api/v1/projects/mine` - Get projects by owner
- ✅ `PUT /api/v1/projects/update` - Update project
- ✅ `PUT /api/v1/projects/update-status` - Update project status
- ✅ `PUT /api/v1/projects/assign-supervisor` - Assign supervisor

**Missing Endpoints:**
- `GET /api/v1/projects` - Get all projects (with optional filters: `status`, `partnerId`, `universityId`)
- `GET /api/v1/projects/:id` - Get project by ID
- `DELETE /api/v1/projects/:id` - Delete project
- `POST /api/v1/projects/upload` - Upload project files (FormData with `files` field)

### 4. Organization Module (`/api/v1/org`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /api/v1/org` - Create organization
- ✅ `GET /api/v1/org?type=university|company` - Get organizations by type

**Missing Endpoints:**
- `GET /api/v1/org/:id` - Get organization by ID
- `PUT /api/v1/org/:id` - Update organization
- `DELETE /api/v1/org/:id` - Delete organization

### 5. Department Module (`/api/v1/departments`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /api/v1/departments` - Create department
- ✅ `GET /api/v1/departments` - Get departments by organization

**Missing Endpoints:**
- `GET /api/v1/departments/:id` - Get department by ID
- `PUT /api/v1/departments/:id` - Update department
- `DELETE /api/v1/departments/:id` - Delete department

### 6. Course Module (`/api/v1/courses`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /api/v1/courses` - Create course
- ✅ `GET /api/v1/courses?department=...` - Get courses by department

**Missing Endpoints:**
- `GET /api/v1/courses/:id` - Get course by ID
- `PUT /api/v1/courses/:id` - Update course
- `DELETE /api/v1/courses/:id` - Delete course

### 7. Milestone Module (`/api/v1/milestones`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /api/v1/milestones` - Create milestone
- ✅ `PUT /api/v1/milestones/update-status` - Update milestone status

**Missing Endpoints:**
- `GET /api/v1/milestones` - Get all milestones (with optional query param: `projectId`)
- `GET /api/v1/milestones/:id` - Get milestone by ID
- `PUT /api/v1/milestones/:id` - Update milestone
- `DELETE /api/v1/milestones/:id` - Delete milestone

### 8. Notification Module (`/api/v1/notifications`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `GET /api/v1/notifications` - Get all notifications for current user
- ✅ `POST /api/v1/notifications` - Create notification
- ✅ `PUT /api/v1/notifications/:notification` - Mark notification as seen

**Missing Endpoints:**
- `GET /api/v1/notifications/:id` - Get notification by ID
- `PATCH /api/v1/notifications/mark-all-read` - Mark all notifications as read (body: `{userId: number}`)

### 9. User Module (`/user`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /user/login` - Login
- ✅ `POST /user/signup` - Signup
- ✅ `GET /user/verify` - Verify token
- ✅ `GET /user/` - Get current user (protected)
- ✅ `POST /user/group` - Create group
- ✅ `POST /user/group/add` - Add member to group
- ✅ `POST /user/group/remove` - Remove member from group

**Missing Endpoints:**
- `GET /user/:id` - Get user by ID
- `GET /user` - Get all users (with optional query param: `role`)
- `PUT /user/:id` - Update user
- `DELETE /user/:id` - Delete user
- `GET /api/v1/users/:id/settings` - Get user settings
- `PUT /api/v1/users/:id/settings` - Update user settings

### 10. Group Module (`/api/v1/groups`)

**Status**: Groups are managed via `/user/group` endpoints, but frontend also expects `/api/v1/groups` endpoints.

**Missing Endpoints:**
- `GET /api/v1/groups` - Get all groups (with optional query params: `courseId`, `userId`)
- `GET /api/v1/groups/:id` - Get group by ID
- `PUT /api/v1/groups/:id` - Update group
- `DELETE /api/v1/groups/:id` - Delete group

### 11. Chat Module (`/api/v1/chats`)

**Status**: Implemented but may need additional endpoints.

**Existing Endpoints:**
- ✅ `GET /api/v1/chats/:group` - Get messages for a group
- ✅ `POST /api/v1/chats` - Create message

**Note**: Frontend also expects:
- `GET /api/v1/chats/threads?userId=...` - Get chat threads for user
- `GET /api/v1/chats/threads/:threadId/messages` - Get messages for thread

### 12. Student Module (`/api/v1/students`)

**Status**: Partially implemented.

**Existing Endpoints:**
- ✅ `POST /api/v1/students` - Create student
- ✅ `GET /api/v1/students?course=...` - Get students by course

**May Need:**
- `GET /api/v1/students/:id` - Get student by ID
- `PUT /api/v1/students/:id` - Update student
- `DELETE /api/v1/students/:id` - Delete student

### 13. Modules Not Yet Implemented

These modules are referenced by frontend but don't exist in backend yet:

**Supervisor Requests:**
- `GET /api/v1/supervisor-requests` - Get requests (with filters: `supervisorId`, `projectId`, `studentId`)
- `GET /api/v1/supervisor-requests/:id` - Get request by ID
- `POST /api/v1/supervisor-requests` - Create request
- `PUT /api/v1/supervisor-requests/:id` - Update request
- `DELETE /api/v1/supervisor-requests/:id` - Delete request
- `GET /api/v1/supervisor/:id/capacity` - Get supervisor capacity

**Portfolio:**
- `GET /api/v1/portfolio` - Get portfolio items (with optional `userId` filter)
- `GET /api/v1/portfolio/:id` - Get portfolio item by ID
- `POST /api/v1/portfolio` - Create portfolio item
- `PUT /api/v1/portfolio/:id` - Update portfolio item
- `DELETE /api/v1/portfolio/:id` - Delete portfolio item

**Invoice:**
- `GET /api/v1/invoices` - Get invoices (with optional `partnerId` filter)
- `GET /api/v1/invoices/:id` - Get invoice by ID
- `GET /api/v1/invoices/:id/download` - Download invoice PDF

**Audit:**
- `GET /api/v1/audit` - Get audit events (with filters: `type`, `actor`, `action`)
- `GET /api/v1/audit/:id` - Get audit event by ID
- `POST /api/v1/audit` - Create audit event

**Proposal (Milestone Proposals):**
- `GET /api/v1/proposals` - Get proposals (with optional `projectId` filter)
- `GET /api/v1/proposals/:id` - Get proposal by ID
- `POST /api/v1/proposals` - Create proposal
- `PUT /api/v1/proposals/:id` - Update proposal
- `DELETE /api/v1/proposals/:id` - Delete proposal

**KYC:**
- `GET /api/v1/kyc` - Get KYC documents (with optional `orgId` filter)
- `GET /api/v1/kyc/:id` - Get KYC document by ID
- `POST /api/v1/kyc` - Create KYC document
- `PUT /api/v1/kyc/:id` - Update KYC document
- `DELETE /api/v1/kyc/:id` - Delete KYC document

**Submission:**
- `GET /api/v1/submissions` - Get submissions (with optional `milestoneId` filter)
- `GET /api/v1/submissions/:id` - Get submission by ID
- `POST /api/v1/submissions` - Create submission
- `PUT /api/v1/submissions/:id` - Update submission
- `DELETE /api/v1/submissions/:id` - Delete submission

**Dispute:**
- `GET /api/v1/disputes` - Get disputes (with filters: `level`, `status`, `raisedBy`)
- `GET /api/v1/disputes/:id` - Get dispute by ID
- `POST /api/v1/disputes` - Create dispute
- `PUT /api/v1/disputes/:id` - Update dispute
- `DELETE /api/v1/disputes/:id` - Delete dispute

## Implementation Priority

### High Priority (Core Functionality)
1. **Application Module** - Critical for student-project matching
2. **Invitation Module** - Needed for user onboarding
3. **Project Module** - Missing GET endpoints
4. **User Module** - Missing user management endpoints

### Medium Priority (Enhanced Features)
5. **Milestone Module** - Missing GET endpoints
6. **Notification Module** - Missing mark-all-read endpoint
7. **Group Module** - Missing GET endpoints
8. **Organization/Department/Course** - Missing GET by ID and update endpoints

### Low Priority (Advanced Features)
9. **Portfolio Module** - For student profiles
10. **Supervisor Requests** - For supervisor assignment
11. **Invoice Module** - For payments
12. **Audit, Proposal, KYC, Submission, Dispute** - For advanced features

## File Upload Implementation

For file upload endpoints (`/upload`), use Fiber's file handling:

```go
import "github.com/gofiber/fiber/v2"

func UploadFiles(c *fiber.Ctx, db *gorm.DB) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"msg": "failed to parse form"})
    }
    
    files := form.File["files"]
    var paths []string
    
    for _, file := range files {
        // Save file
        path := fmt.Sprintf("uploads/%s", file.Filename)
        if err := c.SaveFile(file, path); err != nil {
            return c.Status(400).JSON(fiber.Map{"msg": "failed to save file"})
        }
        paths = append(paths, path)
    }
    
    return c.JSON(fiber.Map{
        "msg": "files uploaded successfully",
        "data": fiber.Map{"paths": paths},
    })
}
```

## Authentication

Most endpoints should use JWT protection:
```go
endpoints := r.Group("/endpoint", user.JWTProtect([]string{"role1", "role2"}))
```

Public endpoints (like invitation accept):
```go
endpoints := r.Group("/endpoint") // No JWT protection
```

## Testing

After implementing endpoints, test with:
1. Postman/Insomnia
2. Frontend integration
3. Verify response format matches `{msg, data}`
4. Test authentication
5. Test error handling

## Notes

- All endpoints should validate input
- Use GORM for database operations
- Follow existing code patterns in other modules
- Return consistent error messages
- Handle edge cases (not found, unauthorized, etc.)













