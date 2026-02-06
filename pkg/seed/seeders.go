package seed

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	college "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/College"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	dispute "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Dispute"
	invitation "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Invitation"
	milestone "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Milestone"
	notification "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Notification"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	portfolio "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Portfolio"
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	student "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Student"
	supervisor "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Supervisor"
	supervisorrequest "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/SupervisorRequest"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func seedOrganizations(db *gorm.DB, count int, passwordHash string) []organization.Organization {
	var orgs []organization.Organization
	// Interleave universities and partners so we always get both (projects need partners)
	for i := 0; i < count; i++ {
		var orgName, orgType string
		if i%2 == 0 {
			orgType = "university"
			orgName = ugandanUniversities[(i/2)%len(ugandanUniversities)]
		} else {
			orgType = "partner"
			orgName = ugandanCompanies[(i/2)%len(ugandanCompanies)]
		}
		adminUser := user.User{
			Role:     "university-admin",
			Email:    fmt.Sprintf("admin@%s.com", sanitizeEmail(orgName)),
			Name:     fmt.Sprintf("%s %s", getRandomFirstName(), getRandomLastName()),
			Password: passwordHash,
			Profile: user.Profile{
				Phone:    fmt.Sprintf("+2567%08d", rand.Intn(100000000)),
				Location: "Kampala, Uganda",
			},
		}
		if orgType == "partner" {
			adminUser.Role = "partner"
		}
		db.Create(&adminUser)
		org := organization.Organization{
			Name:       orgName,
			Type:       orgType,
			IsApproved: true,
			UserID:     adminUser.ID,
			Website:    fmt.Sprintf("https://www.%s.com", sanitizeEmail(orgName)),
			Address:    fmt.Sprintf("%d %s Street, Kampala, Uganda", rand.Intn(999)+1, getRandomStreetName()),
		}
		db.Create(&org)
		orgs = append(orgs, org)
	}
	return orgs
}

func seedBranches(db *gorm.DB, orgs []organization.Organization) []branch.Branch {
	var branches []branch.Branch
	branchNames := []string{"Main Branch", "Kampala Branch", "Entebbe Branch", "Jinja Branch", "Mbarara Branch", "Gulu Branch"}
	for _, org := range orgs {
		n := rand.Intn(2) + 1
		for i := 0; i < n && i < len(branchNames); i++ {
			b := branch.Branch{
				Name:           fmt.Sprintf("%s - %s", org.Name, branchNames[(len(branches)+i)%len(branchNames)]),
				OrganizationID: uint(org.ID),
			}
			db.Create(&b)
			branches = append(branches, b)
		}
	}
	return branches
}

func seedColleges(db *gorm.DB, orgs []organization.Organization) []college.College {
	var colleges []college.College
	collegeNames := []string{"College of Engineering", "College of Business", "College of Sciences", "College of Humanities", "College of Health Sciences"}
	for _, org := range orgs {
		if org.Type != "university" {
			continue
		}
		n := rand.Intn(3) + 1
		for i := 0; i < n && i < len(collegeNames); i++ {
			c := college.College{
				Name:           collegeNames[(len(colleges)+i)%len(collegeNames)],
				OrganizationID: uint(org.ID),
			}
			db.Create(&c)
			colleges = append(colleges, c)
		}
	}
	return colleges
}

func seedDepartments(db *gorm.DB, orgs []organization.Organization, colleges []college.College) []department.Department {
	var depts []department.Department
	orgColleges := make(map[uint][]college.College)
	for _, c := range colleges {
		orgColleges[c.OrganizationID] = append(orgColleges[c.OrganizationID], c)
	}
	for _, org := range orgs {
		if org.Type != "university" {
			continue
		}
		numDepts := rand.Intn(6) + 3
		offset := (int(org.ID) * 7) % len(ugandanDepartments)
		for i := 0; i < numDepts && i < len(ugandanDepartments); i++ {
			idx := (offset + i) % len(ugandanDepartments)
			dept := department.Department{
				Name:           ugandanDepartments[idx],
				OrganizationID: uint(org.ID),
			}
			if cols := orgColleges[uint(org.ID)]; len(cols) > 0 && rand.Float32() < 0.5 {
				dept.CollegeID = &cols[rand.Intn(len(cols))].ID
			}
			db.Create(&dept)
			depts = append(depts, dept)
		}
	}
	return depts
}

func seedCourses(db *gorm.DB, depts []department.Department) []course.Course {
	var courses []course.Course
	for _, dept := range depts {
		numCourses := rand.Intn(4) + 2
		for i := 0; i < numCourses && i < len(ugandanCourses); i++ {
			crs := course.Course{
				Name:         ugandanCourses[(len(courses)+i)%len(ugandanCourses)],
				DepartmentID: uint(dept.ID),
			}
			db.Create(&crs)
			courses = append(courses, crs)
		}
	}
	return courses
}

func seedUsers(db *gorm.DB, orgs []organization.Organization, courses []course.Course, passwordHash string, count int) []user.User {
	var users []user.User
	roles := []string{"student", "supervisor", "partner"}
	universityOrgs := []organization.Organization{}
	for _, o := range orgs {
		if o.Type == "university" {
			universityOrgs = append(universityOrgs, o)
		}
	}
	for i := 0; i < count; i++ {
		role := roles[rand.Intn(len(roles))]
		var courseID uint
		var orgID *uint
		if role == "student" && len(courses) > 0 {
			courseID = uint(courses[rand.Intn(len(courses))].ID)
		}
		if role == "supervisor" && len(universityOrgs) > 0 {
			uo := universityOrgs[rand.Intn(len(universityOrgs))]
			orgID = &uo.ID
		}
		firstName := getRandomFirstName()
		lastName := getRandomLastName()
		email := fmt.Sprintf("%s.%s.%d@example.com", sanitizeEmail(firstName), sanitizeEmail(lastName), i+1000)
		userSkills := []string{}
		for j := 0; j < rand.Intn(5)+3; j++ {
			userSkills = append(userSkills, skills[rand.Intn(len(skills))])
		}
		skillsJSON, _ := json.Marshal(userSkills)
		usr := user.User{
			Role:     role,
			Email:    email,
			Name:     fmt.Sprintf("%s %s", firstName, lastName),
			Password: passwordHash,
			Profile: user.Profile{
				Phone:    fmt.Sprintf("+2567%08d", rand.Intn(100000000)),
				Location: "Kampala, Uganda",
				Bio:      fmt.Sprintf("Experienced %s with expertise in various technologies", role),
				Skills:   datatypes.JSON(skillsJSON),
			},
			CourseID: courseID,
		}
		if orgID != nil {
			usr.OrgID = orgID
		}
		db.Create(&usr)
		users = append(users, usr)
	}
	return users
}

func seedStudents(db *gorm.DB, users []user.User, courses []course.Course) []student.Student {
	var students []student.Student
	for _, usr := range users {
		if usr.Role == "student" && usr.CourseID > 0 {
			std := student.Student{UserID: usr.ID, CourseID: uint(usr.CourseID)}
			db.Create(&std)
			students = append(students, std)
		}
	}
	return students
}

func seedSupervisors(db *gorm.DB, users []user.User, depts []department.Department) []supervisor.Supervisor {
	var supervisors []supervisor.Supervisor
	for _, usr := range users {
		if usr.Role != "supervisor" || usr.OrgID == nil {
			continue
		}
		for _, dept := range depts {
			if dept.OrganizationID == *usr.OrgID {
				sup := supervisor.Supervisor{UserID: usr.ID, DepartmentID: dept.ID}
				db.Create(&sup)
				supervisors = append(supervisors, sup)
				break
			}
		}
	}
	return supervisors
}

func seedProjects(db *gorm.DB, orgs []organization.Organization, depts []department.Department, supervisors []supervisor.Supervisor, count int) []project.Project {
	var projects []project.Project
	partnerUserIDs := []uint{}
	for _, o := range orgs {
		if o.Type == "partner" {
			partnerUserIDs = append(partnerUserIDs, o.UserID)
		}
	}
	if len(partnerUserIDs) == 0 || len(depts) == 0 {
		return projects
	}
	statuses := []string{"draft", "published", "in-progress", "completed", "on-hold"}
	currencies := []string{"USD", "UGX"}
	for i := 0; i < count; i++ {
		partnerUserID := partnerUserIDs[rand.Intn(len(partnerUserIDs))]
		dept := depts[rand.Intn(len(depts))]
		title := projectTitles[i%len(projectTitles)]
		description := projectDescriptions[i%len(projectDescriptions)]
		projectSkills := []string{}
		for j := 0; j < rand.Intn(5)+3; j++ {
			projectSkills = append(projectSkills, skills[rand.Intn(len(skills))])
		}
		skillsJSON, _ := json.Marshal(projectSkills)
		currency := currencies[rand.Intn(len(currencies))]
		budgetValue := uint(rand.Intn(50000) + 1000)
		if currency == "UGX" {
			budgetValue = uint(rand.Intn(200000000) + 1000000)
		}
		deadline := time.Now().AddDate(0, 0, rand.Intn(150)+30).Format("2006-01-02")
		var supervisorID *uint
		if len(supervisors) > 0 && rand.Float32() < 0.3 {
			sup := supervisors[rand.Intn(len(supervisors))]
			if sup.DepartmentID == dept.ID {
				supervisorID = &sup.UserID
			}
		}
		teamStructures := []string{"individuals", "groups", "both"}
		durations := []string{"8 weeks", "12 weeks", "16 weeks", "3 months", "6 months"}
		proj := project.Project{
			DepartmentID:       int(dept.ID),
			Title:              title,
			Description:        description,
			Summary:            fmt.Sprintf("Project summary: %s. Delivers value through %s.", title, getRandomFeature()),
			TeamStructure:      teamStructures[rand.Intn(len(teamStructures))],
			Duration:           durations[rand.Intn(len(durations))],
			Skills:             datatypes.JSON(skillsJSON),
			Budget:             project.Budget{Currency: currency, Value: budgetValue},
			Deadline:           deadline,
			Capacity:           uint(rand.Intn(5) + 1),
			Status:             statuses[rand.Intn(len(statuses))],
			UserID:             partnerUserID,
			SupervisorID:       supervisorID,
		}
		db.Create(&proj)
		projects = append(projects, proj)
	}
	return projects
}

func seedApplications(db *gorm.DB, users []user.User, projects []project.Project, count int) []application.Application {
	var applications []application.Application
	studentUsers := []user.User{}
	for _, u := range users {
		if u.Role == "student" {
			studentUsers = append(studentUsers, u)
		}
	}
	if len(studentUsers) == 0 || len(projects) == 0 {
		return applications
	}
	statusWeights := map[string]int{"SUBMITTED": 40, "SHORTLISTED": 20, "OFFERED": 15, "ASSIGNED": 10, "DECLINED": 5, "WAITLIST": 5, "REJECTED": 5}
	getStatus := func() string {
		total := 0
		for _, w := range statusWeights {
			total += w
		}
		r := rand.Intn(total)
		c := 0
		for s, w := range statusWeights {
			c += w
			if r < c {
				return s
			}
		}
		return "SUBMITTED"
	}
	for i := 0; i < count; i++ {
		student := studentUsers[rand.Intn(len(studentUsers))]
		proj := projects[rand.Intn(len(projects))]
		studentIDs := []uint{student.ID}
		studentIDsJSON, _ := json.Marshal(studentIDs)
		status := getStatus()
		app := application.Application{
			ProjectID:     proj.ID,
			ApplicantType: "INDIVIDUAL",
			StudentIDs:    datatypes.JSON(studentIDsJSON),
			Statement:     fmt.Sprintf("I am interested in this project. I have experience in %s.", getRandomSkills(3)),
			Status:        status,
		}
		if status == "OFFERED" {
			exp := datatypes.Date(time.Now().AddDate(0, 0, rand.Intn(7)+7))
			app.OfferExpiresAt = &exp
		}
		db.Create(&app)
		applications = append(applications, app)
	}
	return applications
}

func seedMilestones(db *gorm.DB, projects []project.Project) []milestone.Milestone {
	var milestones []milestone.Milestone
	statuses := []string{"PROPOSED", "IN_PROGRESS", "SUBMITTED", "APPROVED", "RELEASED", "COMPLETED"}
	for _, proj := range projects {
		for i := 0; i < rand.Intn(4)+2; i++ {
			mil := milestone.Milestone{
				ProjectID:          proj.ID,
				Title:              fmt.Sprintf("Milestone %d: %s", i+1, getRandomMilestoneTitle()),
				Scope:              fmt.Sprintf("Complete %s functionality", getRandomFeature()),
				AcceptanceCriteria: "All tests passing, code reviewed",
				DueDate:            time.Now().AddDate(0, 0, rand.Intn(90)+30).Format("2006-01-02"),
				Amount:             rand.Intn(10000) + 500,
				Currency:           "USD",
				Status:             statuses[rand.Intn(len(statuses))],
			}
			db.Create(&mil)
			milestones = append(milestones, mil)
		}
	}
	return milestones
}

func seedPortfolio(db *gorm.DB, users []user.User, projects []project.Project, milestones []milestone.Milestone) []portfolio.PortfolioItem {
	var items []portfolio.PortfolioItem
	complexities := []string{"LOW", "MEDIUM", "HIGH"}
	for _, mil := range milestones {
		if mil.Status != "COMPLETED" && mil.Status != "RELEASED" {
			continue
		}
		var proj project.Project
		if db.First(&proj, mil.ProjectID).Error != nil {
			continue
		}
		var apps []application.Application
		db.Where("project_id = ? AND status = ?", mil.ProjectID, "ASSIGNED").Find(&apps)
		for _, app := range apps {
			var studentIDs []uint
			json.Unmarshal(app.StudentIDs, &studentIDs)
			for _, sid := range studentIDs {
				var usr user.User
				if db.First(&usr, sid).Error != nil {
					continue
				}
				complexity := complexities[rand.Intn(len(complexities))]
				if mil.Amount > 5000 {
					complexity = "HIGH"
				} else if mil.Amount < 1000 {
					complexity = "LOW"
				}
				pi := portfolio.PortfolioItem{
					UserID:          usr.ID,
					ProjectID:       proj.ID,
					MilestoneID:     &mil.ID,
					Role:            "Developer",
					Scope:           mil.Scope,
					Proof:           datatypes.JSON([]byte("[]")),
					Rating:          getRandomRating(),
					Complexity:      complexity,
					AmountDelivered: float64(mil.Amount),
					Currency:        mil.Currency,
					OnTime:          rand.Float32() < 0.8,
					VerifiedAt:      datatypes.Date(time.Now().Truncate(24 * time.Hour)),
				}
				db.Create(&pi)
				items = append(items, pi)
			}
		}
	}
	return items
}

func seedGroups(db *gorm.DB, users []user.User, count int) []user.Group {
	var groups []user.Group
	studentUsers := []user.User{}
	for _, u := range users {
		if u.Role == "student" {
			studentUsers = append(studentUsers, u)
		}
	}
	if len(studentUsers) < 2 {
		return groups
	}
	names := []string{"Team Alpha", "Team Beta", "Team Gamma", "Code Warriors", "Tech Titans", "Dev Squad"}
	for i := 0; i < count && i < len(names); i++ {
		leader := studentUsers[rand.Intn(len(studentUsers))]
		others := []user.User{}
		for _, s := range studentUsers {
			if s.ID != leader.ID {
				others = append(others, s)
			}
		}
		memberCount := rand.Intn(min(3, len(others))) + 1
		members := []user.User{leader}
		used := map[uint]bool{leader.ID: true}
		for j := 0; j < memberCount && j < len(others); j++ {
			o := others[rand.Intn(len(others))]
			if !used[o.ID] {
				used[o.ID] = true
				members = append(members, o)
			}
		}
		g := user.Group{UserID: leader.ID, Name: names[i%len(names)], Capacity: len(members) + rand.Intn(3)}
		db.Create(&g)
		db.Model(&g).Association("Members").Append(members)
		groups = append(groups, g)
	}
	return groups
}

func seedChatMessages(db *gorm.DB, groups []user.Group, users []user.User, count int) []chat.Message {
	var messages []chat.Message
	if len(groups) == 0 {
		return messages
	}
	templates := []string{
		"Let's discuss the project timeline.", "I've completed the initial design.",
		"Can we schedule a meeting?", "The backend API is ready.", "Great work on the frontend!",
	}
	for i := 0; i < count; i++ {
		g := groups[rand.Intn(len(groups))]
		var gwm user.Group
		db.Preload("Members").First(&gwm, g.ID)
		if len(gwm.Members) == 0 {
			continue
		}
		sender := gwm.Members[rand.Intn(len(gwm.Members))]
		msg := chat.Message{
			SenderID: sender.ID,
			GroupID:  g.ID,
			Body:     templates[rand.Intn(len(templates))],
		}
		msg.CreatedAt = time.Now().AddDate(0, 0, -rand.Intn(30)).Add(-time.Duration(rand.Intn(24)) * time.Hour)
		db.Create(&msg)
		messages = append(messages, msg)
	}
	return messages
}

func seedDisputes(db *gorm.DB, users []user.User, count int) []dispute.Dispute {
	var disputes []dispute.Dispute
	if len(users) < 2 {
		return disputes
	}
	statuses := []string{"pending", "under_review", "resolved", "dismissed"}
	reasons := []string{"Payment not received", "Work quality dispute", "Timeline disagreement", "Scope mismatch"}
	for i := 0; i < count; i++ {
		issuer := users[rand.Intn(len(users))]
		defendants := []user.User{}
		for _, u := range users {
			if u.ID != issuer.ID {
				defendants = append(defendants, u)
			}
		}
		if len(defendants) == 0 {
			continue
		}
		def := defendants[rand.Intn(len(defendants))]
		d := dispute.Dispute{
			SubjectType: "project",
			Reason:      reasons[rand.Intn(len(reasons))],
			Description: fmt.Sprintf("Dispute regarding %s", reasons[rand.Intn(len(reasons))]),
			Status:      statuses[rand.Intn(len(statuses))],
			Level:       "medium",
			IssuerID:    issuer.ID,
			DefendantID: def.ID,
		}
		db.Create(&d)
		disputes = append(disputes, d)
	}
	return disputes
}

func seedInvitations(db *gorm.DB, orgs []organization.Organization, depts []department.Department, count int) []invitation.Invitation {
	var invitations []invitation.Invitation
	statuses := []string{"PENDING", "USED", "EXPIRED"}
	for i := 0; i < count; i++ {
		org := orgs[rand.Intn(len(orgs))]
		role := "student"
		if rand.Float32() < 0.3 {
			role = "supervisor"
		}
		var deptID *uint
		if role == "supervisor" {
			for _, d := range depts {
				if d.OrganizationID == org.ID {
					deptID = &d.ID
					break
				}
			}
		}
		token, _ := invitation.GenerateToken()
		if token == "" {
			token = fmt.Sprintf("token_%d_%d", time.Now().UnixNano(), i)
		}
		status := statuses[rand.Intn(len(statuses))]
		inv := invitation.Invitation{
			Email:          fmt.Sprintf("invite_%s_%d@example.com", role, i),
			Name:           fmt.Sprintf("%s %s", getRandomFirstName(), getRandomLastName()),
			Role:           role,
			OrganizationID: org.ID,
			DepartmentID:   deptID,
			Token:          token,
			Status:         status,
			ExpiresAt:      time.Now().AddDate(0, 0, 7+rand.Intn(14)),
		}
		if status == "EXPIRED" {
			inv.ExpiresAt = time.Now().AddDate(0, 0, -(rand.Intn(7) + 1))
		}
		db.Create(&inv)
		invitations = append(invitations, inv)
	}
	return invitations
}

func seedNotifications(db *gorm.DB, users []user.User, count int) []notification.Notification {
	var notifications []notification.Notification
	types := []string{"application_status", "offer_received", "milestone_approved", "project_assigned", "message_received"}
	titles := map[string]string{
		"application_status": "Application Status Updated",
		"offer_received":      "New Offer Received",
		"milestone_approved":  "Milestone Approved",
		"project_assigned":    "Project Assigned",
		"message_received":    "New Message",
	}
	for i := 0; i < count; i++ {
		u := users[rand.Intn(len(users))]
		t := types[rand.Intn(len(types))]
		n := notification.Notification{
			Type: t, Title: titles[t], Message: "Notification message.",
			Seen: rand.Float32() < 0.7, Link: "/", UserID: u.ID,
		}
		n.CreatedAt = time.Now().AddDate(0, 0, -rand.Intn(30))
		db.Create(&n)
		notifications = append(notifications, n)
	}
	return notifications
}

func seedSupervisorRequests(db *gorm.DB, projects []project.Project, users []user.User, groups []user.Group, count int) []supervisorrequest.SupervisorRequest {
	var requests []supervisorrequest.SupervisorRequest
	students := []user.User{}
	supervisors := []user.User{}
	for _, u := range users {
		if u.Role == "student" {
			students = append(students, u)
		} else if u.Role == "supervisor" {
			supervisors = append(supervisors, u)
		}
	}
	if len(students) == 0 || len(supervisors) == 0 || len(projects) == 0 {
		return requests
	}
	statuses := []string{"PENDING", "APPROVED", "DENIED"}
	for i := 0; i < count; i++ {
		proj := projects[rand.Intn(len(projects))]
		sup := supervisors[rand.Intn(len(supervisors))]
		student := students[rand.Intn(len(students))]
		r := supervisorrequest.SupervisorRequest{
			ProjectID:        proj.ID,
			StudentOrGroupID: student.ID,
			SupervisorID:     sup.ID,
			Status:           statuses[rand.Intn(len(statuses))],
			Message:          "I would like to request your supervision.",
		}
		r.CreatedAt = time.Now().AddDate(0, 0, -rand.Intn(60))
		db.Create(&r)
		requests = append(requests, r)
	}
	return requests
}

func seedDelegatedAccesses(db *gorm.DB, orgs []organization.Organization, users []user.User) []delegatedaccess.DelegatedAccess {
	var accesses []delegatedaccess.DelegatedAccess
	universityAdmins := []user.User{}
	for _, o := range orgs {
		if o.Type != "university" {
			continue
		}
		var u user.User
		if db.First(&u, o.UserID).Error == nil {
			universityAdmins = append(universityAdmins, u)
		}
	}
	otherUsers := []user.User{}
	for _, u := range users {
		if u.Role == "supervisor" || u.Role == "student" {
			otherUsers = append(otherUsers, u)
		}
	}
	if len(universityAdmins) == 0 || len(otherUsers) < 2 {
		return accesses
	}
	n := min(15, len(universityAdmins)*2)
	for i := 0; i < n; i++ {
		delegator := universityAdmins[rand.Intn(len(universityAdmins))]
		delegated := otherUsers[rand.Intn(len(otherUsers))]
		if delegated.ID == delegator.ID {
			continue
		}
		var org organization.Organization
		if db.Where("user_id = ?", delegator.ID).First(&org).Error != nil {
			continue
		}
		da := delegatedaccess.DelegatedAccess{
			DelegatedUserID: delegated.ID,
			DelegatorID:     delegator.ID,
			OrganizationID:  org.ID,
			IsActive:        rand.Float32() < 0.85,
		}
		db.Create(&da)
		accesses = append(accesses, da)
	}
	return accesses
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getRandomFirstName() string   { return ugandanFirstNames[rand.Intn(len(ugandanFirstNames))] }
func getRandomLastName() string    { return ugandanLastNames[rand.Intn(len(ugandanLastNames))] }
func getRandomMilestoneTitle() string {
	titles := []string{"User Auth", "Database Setup", "API Development", "Frontend", "Testing", "Deployment"}
	return titles[rand.Intn(len(titles))]
}
func getRandomFeature() string {
	features := []string{"user management", "payment processing", "reporting", "notifications", "search"}
	return features[rand.Intn(len(features))]
}
func getRandomSkills(n int) string {
	var s []string
	for i := 0; i < n && i < len(skills); i++ {
		s = append(s, skills[rand.Intn(len(skills))])
	}
	return fmt.Sprintf("%v", s)
}
func getRandomRating() *float64 {
	r := float64(rand.Intn(3)+3) + rand.Float64()
	return &r
}
func sanitizeEmail(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		}
	}
	return result
}
func getRandomStreetName() string {
	streets := []string{"Kampala Road", "Nakasero Road", "Kololo Hill", "Muyenga Road", "Ntinda Road", "Entebbe Road"}
	return streets[rand.Intn(len(streets))]
}
