package virtualfs

import (
	"time"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type NodeKind string

const (
	NodeKindFile      NodeKind = "file"
	NodeKindDirectory NodeKind = "directory"
)

type NodeStatus string

const (
	NodeStatusPending NodeStatus = "pending"
	NodeStatusReady   NodeStatus = "ready"
)

type Node struct {
	bun.BaseModel `bun:"vfs_nodes"`

	ID        uuid.UUID  `bun:",pk,type:uuid"`
	PublicID  string     `bun:"public_id,notnull"`
	AccountID uuid.UUID  `bun:"account_id,notnull,type:uuid"`
	ParentID  uuid.UUID  `bun:"parent_id"`
	Kind      NodeKind   `bun:"kind,notnull"`
	Status    NodeStatus `bun:"status,notnull"`
	Name      string     `bun:"name,notnull"`

	BlobKey  blob.Key `bun:"blob_key"`
	Size     int64    `bun:"size"`
	MimeType string   `bun:"mime_type"`

	CreatedAt time.Time `bun:"created_at,notnull,nullzero"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero"`
	DeletedAt time.Time `bun:"deleted_at"`
}

// IsAccessible returns true if the node can be accessed.
// If the node is not ready or if it is soft deleted, it cannot be accessed.
func (n *Node) IsAccessible() bool {
	return n.DeletedAt.IsZero() && n.Status == NodeStatusReady
}
