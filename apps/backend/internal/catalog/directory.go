package catalog

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

const (
	DirItemKindDirectory = "directory"
	DirItemKindFile      = "file"
)

// DirectoryInfo represents directory metadata
// @Description Directory information including path and timestamps
type DirectoryInfo struct {
	// Item type, always "directory"
	Kind string `json:"kind" example:"directory"`
	// Unique directory identifier
	ID string `json:"id" example:"kRp2XYTq9A55"`
	// ParentID is the public ID of the directory this directory is in
	ParentID string `json:"parentId,omitempty" example:"kRp2XYTq9A55"`
	// Full path from root (included when ?include=path)
	Path virtualfs.Path `json:"path,omitempty"`
	// Directory name
	Name string `json:"name" example:"My Documents"`
	// When the directory was created (ISO 8601)
	CreatedAt time.Time `json:"createdAt" example:"2024-12-13T15:04:05Z"`
	// When the directory was last updated (ISO 8601)
	UpdatedAt time.Time `json:"updatedAt" example:"2024-12-13T16:30:00Z"`
	// When the directory was trashed, null if not trashed (ISO 8601)
	DeletedAt *time.Time `json:"deletedAt,omitempty" example:"2024-12-14T10:00:00Z"`
}

// createDirectoryRequest represents a new directory creation request
// @Description Request to create a new directory
type createDirectoryRequest struct {
	// ID of the parent directory
	ParentID string `json:"parentID" example:"kRp2XYTq9A55"`
	// Name for the new directory
	Name string `json:"name" example:"New Folder"`
}

// postDirectoryContentRequest represents a move items request
// @Description Request to move items into this directory
type postDirectoryContentRequest struct {
	// Array of file/directory IDs to move
	Items []string `json:"items" example:"mElnUNCm8F22,kRp2XYTq9A55"`
}

// listDirectoryResponse represents the response to a request to list the contents of a directory
// @Description Response to a request to list the contents of a directory
type listDirectoryResponse struct {
	// Items is the list of items in the directory, limited to the limit specified in the request
	Items []any `json:"items"`
	// NextCursor is the cursor to use to get the next page of results
	NextCursor string `json:"nextCursor,omitempty"`
}

// moveItemsToDirectoryResponse represents the response to a request
// to move items into a directory.
// @Description Response from moving items to a directory with status for each item
type moveItemsToDirectoryResponse struct {
	// Array of items included in the request (FileInfo or DirectoryInfo objects)
	Items []any `json:"items"`

	// Array of IDs of successfully moved items
	Moved []string `json:"moved" example:"mElnUNCm8F22,kRp2XYTq9A55"`

	// Array of IDs of items that conflicted with existing items in the target directory
	Conflicts []string `json:"conflicts" example:"xYz123AbC456"`

	// Array of errors that occurred during the move operation
	Errors []moveItemError `json:"errors"`
}

// moveItemError represents an error that occurred while moving a specific item
// @Description Error details for a failed item move
type moveItemError struct {
	// ID of the item that failed to move
	ID string `json:"id" example:"mElnUNCm8F22"`
	// Error message describing what went wrong
	Error string `json:"error" example:"permission denied"`
}

type decodedListChildrenCursor struct {
	orderBy        virtualfs.ListChildrenOrder
	orderDirection virtualfs.ListChildrenDirection
	nodeID         string
}

func (h *HTTPHandler) currentDirectoryMiddleware(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	directoryID := c.Params("directoryID")

	var node *virtualfs.Node
	if directoryID == "root" {
		n, err := h.vfs.FindRootDirectory(c.Context(), h.db, account.ID)
		if err != nil {
			return httperr.Internal(err)
		}
		node = n
	} else {
		n, err := h.vfs.FindNodeByPublicID(c.Context(), h.db, account.ID, directoryID)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}
		node = n
	}

	c.Locals("directory", node)

	return c.Next()
}

func mustCurrentDirectoryNode(c *fiber.Ctx) *virtualfs.Node {
	return c.Locals("directory").(*virtualfs.Node)
}

func includeParam(c *fiber.Ctx) []string {
	return strings.Split(c.Query("include"), ",")
}

// createDirectory creates a new directory
// @Summary Create directory
// @Description Create a new directory within a parent directory
// @Tags directories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param request body createDirectoryRequest true "Directory details"
// @Param include query string false "Include additional fields" Enums(path)
// @Success 200 {object} DirectoryInfo "Created directory"
// @Failure 400 {object} map[string]string "Parent not found or not a directory"
// @Failure 401 {string} string "Not authenticated"
// @Failure 409 {object} map[string]string "Directory already exists"
// @Router /accounts/{accountID}/directories [post]
func (h *HTTPHandler) createDirectory(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	req := new(createDirectoryRequest)
	if err := c.BodyParser(req); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	parent, err := h.vfs.FindNodeByPublicID(c.Context(), tx, account.ID, req.ParentID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Parent not found"})
		}
		return httperr.Internal(err)
	}

	if parent.Kind != virtualfs.NodeKindDirectory {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Parent is not a directory"})
	}

	node, err := h.vfs.CreateDirectory(c.Context(), tx, account.ID, parent.ID, req.Name)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Directory already exists"})
		}
		return httperr.Internal(err)
	}

	i := DirectoryInfo{
		Kind:      DirItemKindDirectory,
		ID:        node.PublicID,
		Name:      node.Name,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	}

	include := includeParam(c)
	if slices.Contains(include, "path") {
		p, err := h.vfs.RealPath(c.Context(), tx, node)
		if err != nil {
			return httperr.Internal(err)
		}
		i.Path = p
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(i)
}

// fetchDirectory returns directory metadata
// @Summary Get directory info
// @Description Retrieve metadata for a specific directory
// @Tags directories
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Directory ID"
// @Param include query string false "Include additional fields" Enums(path)
// @Success 200 {object} DirectoryInfo "Directory metadata"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Directory not found"
// @Router /accounts/{accountID}/directories/{directoryID} [get]
func (h *HTTPHandler) fetchDirectory(c *fiber.Ctx) error {
	node := mustCurrentDirectoryNode(c)

	i := DirectoryInfo{
		Kind:      DirItemKindDirectory,
		ID:        node.PublicID,
		Name:      node.Name,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	}

	include := includeParam(c)
	if slices.Contains(include, "path") {
		p, err := h.vfs.RealPath(c.Context(), h.db, node)
		if err != nil {
			return httperr.Internal(err)
		}
		i.Path = p
	}

	return c.JSON(i)
}

// listDirectory returns directory contents
// @Summary List directory contents
// @Description Get all files and subdirectories within a directory with optional pagination, sorting, and filtering
// @Tags directories
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Directory ID (use 'root' for the root directory)"
// @Param orderBy query string false "Sort field: name, createdAt, or updatedAt" Enums(name,createdAt,updatedAt)
// @Param dir query string false "Sort direction: asc or desc" Enums(asc,desc)
// @Param limit query integer false "Maximum number of items to return (default: 100, min: 1)"
// @Param cursor query string false "Cursor for pagination (base64-encoded cursor from previous response)"
// @Success 200 {object} listDirectoryResponse "Paginated list of FileInfo and DirectoryInfo objects"
// @Failure 400 {object} map[string]string "Invalid limit or cursor"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Directory not found"
// @Router /accounts/{accountID}/directories/{directoryID}/content [get]
func (h *HTTPHandler) listDirectory(c *fiber.Ctx) error {
	node := mustCurrentDirectoryNode(c)

	opts := virtualfs.ListChildrenOptions{}

	if by := c.Query("orderBy"); by != "" {
		switch by {
		case "name":
			opts.OrderBy = virtualfs.ListChildrenOrderByName
		case "createdAt":
			opts.OrderBy = virtualfs.ListChildrenOrderByCreatedAt
		case "updatedAt":
			opts.OrderBy = virtualfs.ListChildrenOrderByUpdatedAt
		}
	}

	if dir := c.Query("dir"); dir != "" {
		switch dir {
		case "asc":
			opts.OrderDirection = virtualfs.ListChildrenDirectionAsc
		case "desc":
			opts.OrderDirection = virtualfs.ListChildrenDirectionDesc
		}
	}

	if limit := c.Query("limit"); limit != "" {
		limit, err := strconv.Atoi(limit)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid limit"})
		}
		if limit < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Limit must be at least 1"})
		}
		opts.Limit = limit
	}

	if cursor := c.Query("cursor"); cursor != "" {
		dc, err := decodeListChildrenCursor(cursor)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid cursor"})
		}

		n, err := h.vfs.FindNodeByPublicID(c.Context(), h.db, node.AccountID, dc.nodeID)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid cursor"})
			}
			return httperr.Internal(err)
		}

		opts.Cursor = &virtualfs.ListChildrenCursor{
			Node:           n,
			OrderBy:        dc.orderBy,
			OrderDirection: dc.orderDirection,
		}
	}

	children, cursor, err := h.vfs.ListChildren(c.Context(), h.db, node, opts)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	items := make([]any, len(children))
	for i, child := range children {
		switch child.Kind {
		case virtualfs.NodeKindDirectory:
			items[i] = DirectoryInfo{
				Kind:      DirItemKindDirectory,
				ID:        child.PublicID,
				Name:      child.Name,
				CreatedAt: child.CreatedAt,
				UpdatedAt: child.UpdatedAt,
				DeletedAt: child.DeletedAt,
			}
		case virtualfs.NodeKindFile:
			items[i] = FileInfo{
				Kind:      DirItemKindFile,
				ID:        child.PublicID,
				ParentID:  node.PublicID,
				Name:      child.Name,
				Size:      child.Size,
				MimeType:  child.MimeType,
				CreatedAt: child.CreatedAt,
				UpdatedAt: child.UpdatedAt,
				DeletedAt: child.DeletedAt,
			}
		}
	}

	return c.JSON(listDirectoryResponse{
		Items:      items,
		NextCursor: encodeListChildrenCursor(cursor),
	})
}

// patchDirectory updates directory properties
// @Summary Update directory
// @Description Update directory properties such as name (rename)
// @Tags directories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Directory ID"
// @Param request body patchDirectoryRequest true "Directory update"
// @Success 200 {object} DirectoryInfo "Updated directory metadata"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Directory not found"
// @Router /accounts/{accountID}/directories/{directoryID} [patch]
func (h *HTTPHandler) patchDirectory(c *fiber.Ctx) error {
	node := mustCurrentDirectoryNode(c)

	patch := new(patchDirectoryRequest)
	if err := c.BodyParser(patch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	if patch.Name != "" {
		err := h.vfs.RenameNode(c.Context(), tx, node, patch.Name)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(DirectoryInfo{
		Kind:      DirItemKindDirectory,
		ID:        node.PublicID,
		Name:      node.Name,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	})
}

// deleteDirectory removes a directory
// @Summary Delete directory
// @Description Delete a directory permanently or move it to trash. Deleting a directory also affects all its contents.
// @Tags directories
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Directory ID"
// @Param trash query bool false "Move to trash instead of permanent delete" default(false)
// @Success 204 {string} string "Directory deleted"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Directory not found"
// @Router /accounts/{accountID}/directories/{directoryID} [delete]
func (h *HTTPHandler) deleteDirectory(c *fiber.Ctx) error {
	node := mustCurrentDirectoryNode(c)

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	shouldTrash := c.Query("trash") == "true"
	if shouldTrash {
		_, err := h.vfs.SoftDeleteNode(c.Context(), tx, node)
		if err != nil {
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.JSON(directoryInfoFromNode(node))
	} else {
		err = h.vfs.PermanentlyDeleteNode(c.Context(), tx, node)
		if err != nil {
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// deleteDirectories removes multiple directories
// @Summary Bulk delete directories
// @Description Delete multiple directories permanently or move them to trash. Deleting directories also affects all their contents. All items must be directories.
// @Tags directories
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param id query string true "Comma-separated list of directory IDs to delete" example:"kRp2XYTq9A55,xYz123AbC456"
// @Param trash query bool false "Move to trash instead of permanent delete" default(false)
// @Success 204 {string} string "Directories deleted"
// @Failure 400 {object} map[string]string "All items must be directories"
// @Failure 401 {string} string "Not authenticated"
// @Router /accounts/{accountID}/directories [delete]
func (h *HTTPHandler) deleteDirectories(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	idq := c.Query("id", "")
	if idq == "" {
		return c.SendStatus(fiber.StatusNoContent)
	}

	ids := strings.Split(idq, ",")
	if len(ids) == 0 {
		return c.SendStatus(fiber.StatusNoContent)
	}

	shouldTrash := c.Query("trash") == "true"

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	nodes, err := h.vfs.FindNodesByPublicID(c.Context(), tx, account.ID, ids)
	if err != nil {
		return httperr.Internal(err)
	}

	if len(nodes) == 0 {
		return c.SendStatus(fiber.StatusNoContent)
	}

	for _, node := range nodes {
		if node.Kind != virtualfs.NodeKindDirectory {
			return httperr.NewHTTPError(fiber.StatusBadRequest, "all items must be directories", nil)
		}
	}

	if shouldTrash {
		deleted, err := h.vfs.SoftDeleteNodes(c.Context(), tx, nodes)
		if err != nil {
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		res := make([]DirectoryInfo, 0, len(deleted))
		for _, node := range deleted {
			res = append(res, directoryInfoFromNode(node))
		}

		return c.JSON(deleted)
	} else {
		for _, node := range nodes {
			err = h.vfs.PermanentlyDeleteNode(c.Context(), tx, node)
			if err != nil {
				return httperr.Internal(err)
			}
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

}

// moveItemsToDirectory moves files and directories into this directory
// @Summary Move items to directory
// @Description Move one or more files or directories into this directory. Returns detailed status for each item including which were successfully moved, which had conflicts, and which encountered errors.
// @Tags directories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Target directory ID"
// @Param request body postDirectoryContentRequest true "Items to move"
// @Success 200 {object} moveItemsToDirectoryResponse "Move operation results with moved, conflict, and error states"
// @Failure 400 {object} map[string]string "Invalid request or items not in same directory"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {object} map[string]string "One or more items not found"
// @Router /accounts/{accountID}/directories/{directoryID}/content [post]
func (h *HTTPHandler) moveItemsToDirectory(c *fiber.Ctx) error {
	acc := account.CurrentAccount(c)
	if acc == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	targetDir := mustCurrentDirectoryNode(c)

	req := new(postDirectoryContentRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if len(req.Items) == 0 {
		return c.SendStatus(fiber.StatusNoContent)
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	nodes, err := h.vfs.FindNodesByPublicID(c.Context(), tx, acc.ID, req.Items)
	if err != nil {
		return httperr.Internal(err)
	}
	if len(nodes) != len(req.Items) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "One or more items not found"})
	}

	// Move all nodes to the target directory
	result, err := h.vfs.MoveNodesInSameDirectory(c.Context(), tx, nodes, targetDir.ID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrUnsupportedOperation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "All items must be in the same directory"})
		}
		if errors.Is(err, virtualfs.ErrNodeConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Name conflict in target directory"})
		}
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	res := moveItemsToDirectoryResponse{
		Items:     make([]any, 0),
		Moved:     make([]string, 0),
		Conflicts: make([]string, 0),
		Errors:    make([]moveItemError, 0),
	}

	for _, node := range result.Moved {
		res.Items = append(res.Items, toDirectoryItem(node))
		res.Moved = append(res.Moved, node.PublicID)
	}
	for _, node := range result.Conflicts {
		res.Items = append(res.Items, toDirectoryItem(node))
		res.Conflicts = append(res.Conflicts, node.PublicID)
	}
	for _, err := range result.Errors {
		res.Errors = append(res.Errors, moveItemError{
			ID:    err.Node.PublicID,
			Error: err.Error.Error(),
		})
	}

	return c.JSON(res)
}

func encodeListChildrenCursor(cursor *virtualfs.ListChildrenCursor) string {
	var by int
	switch cursor.OrderBy {
	case virtualfs.ListChildrenOrderByName:
		by = 0
	case virtualfs.ListChildrenOrderByCreatedAt:
		by = 1
	case virtualfs.ListChildrenOrderByUpdatedAt:
		by = 2
	}

	var d int
	switch cursor.OrderDirection {
	case virtualfs.ListChildrenDirectionAsc:
		d = 0
	case virtualfs.ListChildrenDirectionDesc:
		d = 1
	}

	s := fmt.Sprintf("%d:%d:%s", by, d, cursor.Node.ID)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func decodeListChildrenCursor(s string) (*decodedListChildrenCursor, error) {
	bs, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(bs), ":")
	if len(parts) != 3 {
		return nil, errors.New("invalid cursor")
	}

	c := new(decodedListChildrenCursor)

	switch parts[0] {
	case "0":
		c.orderBy = virtualfs.ListChildrenOrderByName
	case "1":
		c.orderBy = virtualfs.ListChildrenOrderByCreatedAt
	case "2":
		c.orderBy = virtualfs.ListChildrenOrderByUpdatedAt
	default:
		return nil, errors.New("invalid cursor")
	}

	switch parts[1] {
	case "0":
		c.orderDirection = virtualfs.ListChildrenDirectionAsc
	case "1":
		c.orderDirection = virtualfs.ListChildrenDirectionDesc
	default:
		return nil, errors.New("invalid cursor")
	}

	c.nodeID = parts[2]

	return c, nil
}
