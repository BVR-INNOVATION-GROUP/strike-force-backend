package student

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	"gorm.io/gorm"
)

// generateCourseCode generates a course code from course name
// Examples: "Computer Science" -> "CS", "Engineering" -> "ENG", "Medicine" -> "MED"
func generateCourseCode(courseName string) string {
	// Remove common words and get initials
	words := strings.Fields(strings.ToUpper(courseName))
	
	// Filter out common words
	commonWords := map[string]bool{
		"AND": true, "OF": true, "THE": true, "IN": true, "FOR": true,
		"TO": true, "A": true, "AN": true,
	}
	
	var initials []string
	for _, word := range words {
		if !commonWords[word] && len(word) > 0 {
			initials = append(initials, string(word[0]))
		}
	}
	
	// If we have at least 2 initials, use them (max 4)
	if len(initials) >= 2 {
		code := strings.Join(initials[:min(4, len(initials))], "")
		return code
	}
	
	// Fallback: use first 3-4 uppercase letters
	upperName := strings.ToUpper(strings.ReplaceAll(courseName, " ", ""))
	if len(upperName) >= 3 {
		return upperName[:min(4, len(upperName))]
	}
	
	// Last resort: pad with X
	return strings.ToUpper(courseName[:min(3, len(courseName))]) + "X"
}

// generateBranchCode generates a branch code from branch name
// Examples: "Undergraduate" -> "UG", "Postgraduate" -> "PG", "Masters" -> "MA"
func generateBranchCode(branchName string) string {
	upperName := strings.ToUpper(branchName)
	
	// Common mappings
	mappings := map[string]string{
		"UNDERGRADUATE": "UG",
		"POSTGRADUATE":  "PG",
		"MASTERS":       "MA",
		"MASTER":        "MA",
		"PHD":           "PhD",
		"DOCTORATE":     "PhD",
		"DOCTOR":        "PhD",
	}
	
	// Check for exact match
	if code, ok := mappings[upperName]; ok {
		return code
	}
	
	// Check if name contains any of the keywords
	for key, code := range mappings {
		if strings.Contains(upperName, key) {
			return code
		}
	}
	
	// Fallback: use first 2-3 uppercase letters
	words := strings.Fields(upperName)
	if len(words) > 0 {
		firstWord := words[0]
		if len(firstWord) >= 2 {
			return firstWord[:min(3, len(firstWord))]
		}
		return firstWord + "X"
	}
	
	return "00" // Default for no branch
}

// generateDistrictCode generates a 3-letter district code
func generateDistrictCode(district string) string {
	if district == "" {
		return "XXX"
	}
	
	// Remove spaces and convert to uppercase
	cleanDistrict := strings.ToUpper(strings.ReplaceAll(district, " ", ""))
	
	// Take first 3 letters
	if len(cleanDistrict) >= 3 {
		return cleanDistrict[:3]
	}
	
	// Pad with X if needed
	return cleanDistrict + strings.Repeat("X", 3-len(cleanDistrict))
}

// luhnChecksum calculates Luhn algorithm checksum digit
func luhnChecksum(id string) int {
	// Remove non-digit characters for checksum calculation
	digits := regexp.MustCompile(`\D`).ReplaceAllString(id, "")
	
	sum := 0
	alternate := false
	
	// Process from right to left
	for i := len(digits) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(digits[i]))
		
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		
		sum += digit
		alternate = !alternate
	}
	
	// Calculate check digit
	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit
}

// getNextSequenceNumber gets the next sequence number for a given course+branch+year combination
func getNextSequenceNumber(db *gorm.DB, courseID uint, branchID *uint, enrollmentYear int) (int, error) {
	var count int64
	
	query := db.Model(&Student{}).
		Where("course_id = ? AND enrollment_year = ?", courseID, enrollmentYear)
	
	if branchID != nil {
		query = query.Where("branch_id = ?", *branchID)
	} else {
		query = query.Where("branch_id IS NULL")
	}
	
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	
	return int(count) + 1, nil
}

// GenerateStudentID generates a unique student ID in the format:
// [YEAR][COURSE_CODE]-[BRANCH_CODE]-[DISTRICT_CODE]-[SEQUENCE]-[CHECKSUM]
// Example: 2024CS-UG-KMP-0001-5
func GenerateStudentID(db *gorm.DB, course course.Course, branch *branch.Branch, district string, enrollmentYear int) (string, error) {
	// Generate course code
	courseCode := generateCourseCode(course.Name)
	
	// Generate branch code
	var branchCode string
	if branch != nil {
		branchCode = generateBranchCode(branch.Name)
	} else {
		branchCode = "00"
	}
	
	// Generate district code
	districtCode := generateDistrictCode(district)
	
	// Get next sequence number
	var branchID *uint
	if branch != nil {
		branchID = &branch.ID
	}
	
	sequence, err := getNextSequenceNumber(db, course.ID, branchID, enrollmentYear)
	if err != nil {
		return "", fmt.Errorf("failed to get sequence number: %w", err)
	}
	
	// Format sequence as 4 digits
	sequenceStr := fmt.Sprintf("%04d", sequence)
	
	// Build ID without checksum
	yearStr := strconv.Itoa(enrollmentYear)
	idWithoutChecksum := fmt.Sprintf("%s%s-%s-%s-%s", yearStr, courseCode, branchCode, districtCode, sequenceStr)
	
	// Calculate checksum
	checksum := luhnChecksum(idWithoutChecksum)
	
	// Final ID
	studentID := fmt.Sprintf("%s-%d", idWithoutChecksum, checksum)
	
	// Ensure uniqueness (in case of collision, increment sequence)
	maxAttempts := 100
	attempts := 0
	for attempts < maxAttempts {
		var existing Student
		// Only check for non-null student_id values to avoid issues with existing NULL records
		err := db.Where("student_id = ? AND student_id IS NOT NULL", studentID).First(&existing).Error
		
		if err == gorm.ErrRecordNotFound {
			// ID is unique
			return studentID, nil
		}
		
		// If column doesn't exist yet, treat as unique (column will be created by AutoMigrate)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "column") && strings.Contains(errStr, "does not exist") {
				// Column doesn't exist yet, ID is effectively unique
				return studentID, nil
			}
		}
		
		// Collision detected, increment sequence
		sequence++
		sequenceStr = fmt.Sprintf("%04d", sequence)
		idWithoutChecksum = fmt.Sprintf("%s%s-%s-%s-%s", yearStr, courseCode, branchCode, districtCode, sequenceStr)
		checksum = luhnChecksum(idWithoutChecksum)
		studentID = fmt.Sprintf("%s-%d", idWithoutChecksum, checksum)
		attempts++
	}
	
	return studentID, fmt.Errorf("failed to generate unique student ID after %d attempts", maxAttempts)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

