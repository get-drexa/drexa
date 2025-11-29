package upload

import "github.com/get-drexa/drexa/internal/virtualfs"

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Upload struct {
	ID         string          `json:"id"`
	Status     Status          `json:"status"`
	TargetNode *virtualfs.Node `json:"-"`
	UploadURL  string          `json:"uploadUrl"`
}
