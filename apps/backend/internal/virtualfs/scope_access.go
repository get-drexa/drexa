package virtualfs

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// nodeWithinRootQuery checks if a node is within a root by traversing ancestors.
const nodeWithinRootQuery = `WITH RECURSIVE path AS (
    SELECT id, parent_id
    FROM vfs_nodes
    WHERE id = ? AND deleted_at IS NULL
    UNION ALL
    SELECT n.id, n.parent_id
    FROM vfs_nodes n
    JOIN path p ON n.id = p.parent_id
    WHERE n.deleted_at IS NULL
)
SELECT 1 FROM path WHERE id = ? LIMIT 1;`

// nodeWithinRootOrAllowedQuery checks if a node is:
// 1. Within the root node (has root in its ancestry)
// 2. Either directly in the allowed list, OR has an ancestor in the allowed list
// This combines two checks into a single recursive query.
const nodeWithinRootOrAllowedQuery = `WITH RECURSIVE path AS (
    SELECT id, parent_id
    FROM vfs_nodes
    WHERE id = ? AND deleted_at IS NULL
    UNION ALL
    SELECT n.id, n.parent_id
    FROM vfs_nodes n
    JOIN path p ON n.id = p.parent_id
    WHERE n.deleted_at IS NULL
)
SELECT
    EXISTS(SELECT 1 FROM path WHERE id = ?) AS within_root,
    EXISTS(SELECT 1 FROM path WHERE id IN (?)) AS within_allowed;`

// filterNodesWithAllowlistQuery filters multiple nodes in a single query.
// It returns node IDs that are within the root AND have an ancestor (or self) in the allowed list.
const filterNodesWithAllowlistQuery = `WITH RECURSIVE node_paths AS (
    -- Start from all input nodes
    SELECT id AS original_id, id, parent_id
    FROM vfs_nodes
    WHERE id IN (?) AND deleted_at IS NULL
    UNION ALL
    -- Traverse up to ancestors
    SELECT np.original_id, n.id, n.parent_id
    FROM vfs_nodes n
    JOIN node_paths np ON n.id = np.parent_id
    WHERE n.deleted_at IS NULL
)
SELECT DISTINCT original_id
FROM node_paths
WHERE original_id IN (
    -- Nodes that have root in their path
    SELECT original_id FROM node_paths WHERE id = ?
)
AND original_id IN (
    -- Nodes that have an allowed node in their path
    SELECT original_id FROM node_paths WHERE id IN (?)
);`

// filterNodesWithinRootQuery filters multiple nodes that are within the root (no allowlist).
const filterNodesWithinRootQuery = `WITH RECURSIVE node_paths AS (
    SELECT id AS original_id, id, parent_id
    FROM vfs_nodes
    WHERE id IN (?) AND deleted_at IS NULL
    UNION ALL
    SELECT np.original_id, n.id, n.parent_id
    FROM vfs_nodes n
    JOIN node_paths np ON n.id = np.parent_id
    WHERE n.deleted_at IS NULL
)
SELECT DISTINCT original_id
FROM node_paths
WHERE id = ?;`

func isScopeSet(scope *Scope) bool {
	return scope != nil && scope.AccountID != uuid.Nil && scope.RootNodeID != uuid.Nil
}

// canAccessNode checks if the scope permits the operation and allows access to the node.
func (vfs *VirtualFS) canAccessNode(ctx context.Context, db bun.IDB, scope *Scope, op Operation, nodeID uuid.UUID) (bool, error) {
	if !scope.Allows(op) {
		return false, nil
	}
	return vfs.isNodeAllowedByScope(ctx, db, scope, nodeID)
}

func (vfs *VirtualFS) isNodeWithinRoot(ctx context.Context, db bun.IDB, nodeID, rootID uuid.UUID) (bool, error) {
	var exists int
	err := db.NewRaw(nodeWithinRootQuery, nodeID, rootID).Scan(ctx, &exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// isNodeAllowedByScope checks if a node is accessible under the given scope.
// It verifies the node is within the root and (if an allowlist exists) within an allowed subtree.
// Optimized to use a single query when an allowlist is present.
func (vfs *VirtualFS) isNodeAllowedByScope(ctx context.Context, db bun.IDB, scope *Scope, nodeID uuid.UUID) (bool, error) {
	if !isScopeSet(scope) {
		return false, nil
	}

	// Fast path: no allowlist means full subtree access under root
	if len(scope.AllowedNodes) == 0 {
		return vfs.isNodeWithinRoot(ctx, db, nodeID, scope.RootNodeID)
	}

	// Quick check: if nodeID is directly in the allowlist, just check root containment
	if _, ok := scope.AllowedNodes[nodeID]; ok {
		return vfs.isNodeWithinRoot(ctx, db, nodeID, scope.RootNodeID)
	}

	// Single query: build ancestry path and check both root and allowlist membership
	allowedIDs := scopeAllowedNodesList(scope)

	var result struct {
		WithinRoot    bool `bun:"within_root"`
		WithinAllowed bool `bun:"within_allowed"`
	}

	err := db.NewRaw(nodeWithinRootOrAllowedQuery, nodeID, scope.RootNodeID, bun.In(allowedIDs)).Scan(ctx, &result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return result.WithinRoot && result.WithinAllowed, nil
}

// filterNodesByScope filters a list of nodes to only those accessible under the scope.
// Optimized to use a single batch query instead of per-node queries.
func (vfs *VirtualFS) filterNodesByScope(ctx context.Context, db bun.IDB, scope *Scope, nodes []*Node) ([]*Node, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	if !isScopeSet(scope) {
		return nil, nil
	}

	nodeIDs := make([]uuid.UUID, len(nodes))
	nodeMap := make(map[uuid.UUID]*Node, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
		nodeMap[node.ID] = node
	}

	var allowedIDs []uuid.UUID
	var err error

	if len(scope.AllowedNodes) == 0 {
		// No allowlist: just check nodes are within root
		err = db.NewRaw(filterNodesWithinRootQuery, bun.In(nodeIDs), scope.RootNodeID).Scan(ctx, &allowedIDs)
	} else {
		// With allowlist: check nodes are within root AND within an allowed subtree
		allowedNodesList := scopeAllowedNodesList(scope)
		err = db.NewRaw(filterNodesWithAllowlistQuery, bun.In(nodeIDs), scope.RootNodeID, bun.In(allowedNodesList)).Scan(ctx, &allowedIDs)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Node, 0), nil
		}
		return nil, err
	}

	allowed := make([]*Node, 0, len(allowedIDs))
	for _, id := range allowedIDs {
		if node, ok := nodeMap[id]; ok {
			allowed = append(allowed, node)
		}
	}

	return allowed, nil
}

// scopeAllowedNodesList converts the AllowedNodes map to a slice for use in queries.
func scopeAllowedNodesList(scope *Scope) []uuid.UUID {
	if scope == nil || len(scope.AllowedNodes) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(scope.AllowedNodes))
	for id := range scope.AllowedNodes {
		ids = append(ids, id)
	}
	return ids
}
