package catalog

import (
	"errors"
	"slices"
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

func (h *HTTPHandler) currentDirectoryMiddleware(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	directoryID := c.Params("directoryID")
	node, err := h.vfs.FindNodeByPublicID(c.Context(), h.db, account.ID, directoryID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
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
// @Description Get all files and subdirectories within a directory
// @Tags directories
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param directoryID path string true "Directory ID"
// @Success 200 {array} interface{} "Array of FileInfo and DirectoryInfo objects"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Directory not found"
// @Router /accounts/{accountID}/directories/{directoryID}/content [get]
func (h *HTTPHandler) listDirectory(c *fiber.Ctx) error {
	node := mustCurrentDirectoryNode(c)
	children, err := h.vfs.ListChildren(c.Context(), h.db, node)
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
				Name:      child.Name,
				Size:      child.Size,
				MimeType:  child.MimeType,
				CreatedAt: child.CreatedAt,
				UpdatedAt: child.UpdatedAt,
				DeletedAt: child.DeletedAt,
			}
		}
	}

	return c.JSON(items)
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
		err = h.vfs.SoftDeleteNode(c.Context(), h.db, node)
		if err != nil {
			return httperr.Internal(err)
		}
	} else {
		err = h.vfs.PermanentlyDeleteNode(c.Context(), h.db, node)
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

	res := moveItemsToDirectoryResponse{}

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
