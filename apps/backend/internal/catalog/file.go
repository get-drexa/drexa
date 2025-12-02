package catalog

import "time"

type FileInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	MimeType  string     `json:"mimeType"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}
