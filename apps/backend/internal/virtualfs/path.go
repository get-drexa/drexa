package virtualfs

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const absolutePathQuery = `WITH RECURSIVE path AS (
    SELECT id, parent_id, name, 1 as depth
    FROM vfs_nodes WHERE id = ? AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT n.id, n.parent_id, n.name, p.depth + 1
    FROM vfs_nodes n
    JOIN path p ON n.id = p.parent_id
    WHERE n.deleted_at IS NULL
)
SELECT name FROM path 
WHERE EXISTS (SELECT 1 FROM path WHERE parent_id IS NULL)
ORDER BY depth DESC;`

func JoinPath(parts ...string) string {
	return strings.Join(parts, "/")
}

func buildNodeAbsolutePath(ctx context.Context, db bun.IDB, nodeID uuid.UUID) (string, error) {
	var path []string
	err := db.NewRaw(absolutePathQuery, nodeID).Scan(ctx, &path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNodeNotFound
		}
		return "", err
	}
	return JoinPath(path...), nil
}
