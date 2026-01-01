package project

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// extractPublicIDFromURL extracts the public_id from a Cloudinary URL
// Example: https://res.cloudinary.com/dcqtrh3hv/raw/upload/v1765279128/strikeforce/projects/4/mou/en5luj1greatvztwlofq.pdf
// Returns: strikeforce/projects/4/mou/en5luj1greatvztwlofq
func extractPublicIDFromURL(cloudinaryURL string) (string, error) {
	if cloudinaryURL == "" {
		return "", fmt.Errorf("empty Cloudinary URL")
	}

	// Parse the URL
	parsedURL, err := url.Parse(cloudinaryURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	// Cloudinary URLs have format: https://res.cloudinary.com/{cloud_name}/{resource_type}/upload/{version}/{public_id}.{format}
	// or: https://res.cloudinary.com/{cloud_name}/{resource_type}/upload/{public_id}.{format}
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")

	// Find the "upload" segment
	uploadIndex := -1
	for i, part := range pathParts {
		if part == "upload" {
			uploadIndex = i
			break
		}
	}

	if uploadIndex == -1 {
		return "", fmt.Errorf("invalid Cloudinary URL format: upload segment not found")
	}

	// Get everything after "upload"
	// Skip version if present (starts with 'v' followed by numbers)
	partsAfterUpload := pathParts[uploadIndex+1:]
	if len(partsAfterUpload) == 0 {
		return "", fmt.Errorf("invalid Cloudinary URL format: no path after upload")
	}

	// Check if first part is a version (starts with 'v' and is numeric)
	if len(partsAfterUpload) > 0 && strings.HasPrefix(partsAfterUpload[0], "v") {
		partsAfterUpload = partsAfterUpload[1:]
	}

	// Join remaining parts to get public_id
	publicID := strings.Join(partsAfterUpload, "/")

	// Remove file extension if present
	if lastDot := strings.LastIndex(publicID, "."); lastDot != -1 {
		publicID = publicID[:lastDot]
	}

	return publicID, nil
}

// deleteFromCloudinary deletes a file from Cloudinary using the Admin API
func deleteFromCloudinary(publicID string) error {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return fmt.Errorf("Cloudinary credentials not configured")
	}

	// Build the deletion URL
	// Cloudinary Admin API: https://api.cloudinary.com/v1_1/{cloud_name}/resources/{resource_type}/upload/{public_id}
	// For raw files (PDFs), resource_type is "raw"
	deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/raw/upload/%s", cloudName, url.QueryEscape(publicID))

	// Create request with basic auth
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set basic authentication
	req.SetBasicAuth(apiKey, apiSecret)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete from Cloudinary: %v", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if msg, ok := errorResp["error"].(map[string]interface{})["message"].(string); ok {
				return fmt.Errorf("Cloudinary deletion failed: %s", msg)
			}
		}
		return fmt.Errorf("Cloudinary deletion failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DeleteMOUFromCloudinary deletes a MOU PDF from Cloudinary given its URL
func DeleteMOUFromCloudinary(mouURL string) error {
	if mouURL == "" {
		return nil // Nothing to delete
	}

	// Extract public_id from URL
	publicID, err := extractPublicIDFromURL(mouURL)
	if err != nil {
		return fmt.Errorf("failed to extract public_id: %v", err)
	}

	// Delete from Cloudinary
	return deleteFromCloudinary(publicID)
}






