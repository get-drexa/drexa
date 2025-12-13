package upload

import "github.com/get-drexa/drexa/internal/virtualfs"

// Status represents the upload state
// @Description Upload status enumeration
type Status string

const (
	// StatusPending indicates upload is awaiting content
	StatusPending Status = "pending"
	// StatusCompleted indicates upload finished successfully
	StatusCompleted Status = "completed"
	// StatusFailed indicates upload failed
	StatusFailed Status = "failed"
)

// Upload represents a file upload session
// @Description File upload session with status and upload URL
type Upload struct {
	// Unique upload session identifier
	ID string `json:"id" example:"xNq5RVBt3K88"`
	// Current upload status
	Status Status `json:"status" example:"pending" enums:"pending,completed,failed"`
	// Internal target node reference
	TargetNode *virtualfs.Node `json:"-" swaggerignore:"true"`
	// URL to upload file content to
	UploadURL string `json:"uploadUrl" example:"https://api.example.com/api/accounts/550e8400-e29b-41d4-a716-446655440000/uploads/xNq5RVBt3K88/content"`
}
