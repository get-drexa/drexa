package virtualfs

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/uptrace/bun"
)

const absolutePathQuery = `WITH RECURSIVE path AS (
    SELECT id, parent_id, name, 1 as depth
    FROM vfs_nodes WHERE id = ? AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT n.id, n.parent_id, n.name, p.depth + 1
    FROM vfs_nodes n
    JOIN path p ON n.id = p.parent_id
)
SELECT name FROM path 
WHERE EXISTS (SELECT 1 FROM path WHERE parent_id IS NULL)
ORDER BY depth DESC;`

const absolutePathWithPublicIDQuery = `WITH RECURSIVE path AS (
    SELECT id, parent_id, name, public_id, 1 as depth
    FROM vfs_nodes WHERE id = ? AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT n.id, n.parent_id, n.name, n.public_id, p.depth + 1
    FROM vfs_nodes n
    JOIN path p ON n.id = p.parent_id
)
SELECT name, public_id FROM path 
WHERE EXISTS (SELECT 1 FROM path WHERE parent_id IS NULL)
ORDER BY depth DESC;`

// Path represents a path to a node.
type Path []PathSegment

// PathSegment represents a segment of a path, containing the name and public ID of the node.
type PathSegment struct {
	Name     string `bun:"name" json:"name"`
	PublicID string `bun:"public_id" json:"id"`
}

func JoinPath(parts ...string) string {
	return strings.Join(parts, "/")
}

func buildNodeAbsolutePathString(ctx context.Context, db bun.IDB, node *Node) (string, error) {
	var path []string
	err := db.NewRaw(absolutePathQuery, node.ID).Scan(ctx, &path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNodeNotFound
		}
		return "", err
	}
	return JoinPath(path...), nil
}

func buildNoteAbsolutePath(ctx context.Context, db bun.IDB, node *Node) (Path, error) {
	var segments []PathSegment
	err := db.NewRaw(absolutePathWithPublicIDQuery, node.ID).Scan(ctx, &segments)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return segments, nil
}
