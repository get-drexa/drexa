package catalog

import (
	"errors"
	"strings"
	"time"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/sharing"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

// FileInfo represents file metadata
// @Description File information including name, size, and timestamps
type FileInfo struct {
	// Item type, always "file"
	Kind string `json:"kind" example:"file"`
	// Unique file identifier
	ID string `json:"id" example:"mElnUNCm8F22"`
	// ParentID is the public ID of the directory this file is in
	ParentID string `json:"parentId,omitempty" example:"kRp2XYTq9A55"`
	// File name
	Name string `json:"name" example:"document.pdf"`
	// File size in bytes
	Size int64 `json:"size" example:"1048576"`
	// MIME type of the file
	MimeType string `json:"mimeType" example:"application/pdf"`
	// When the file was created (ISO 8601)
	CreatedAt time.Time `json:"createdAt" example:"2024-12-13T15:04:05Z"`
	// When the file was last updated (ISO 8601)
	UpdatedAt time.Time `json:"updatedAt" example:"2024-12-13T16:30:00Z"`
	// When the file was trashed, null if not trashed (ISO 8601)
	DeletedAt *time.Time `json:"deletedAt,omitempty" example:"2024-12-14T10:00:00Z"`
}

func mustCurrentFileNode(c *fiber.Ctx) *virtualfs.Node {
	return c.Locals("file").(*virtualfs.Node)
}

func (h *HTTPHandler) currentFileMiddleware(c *fiber.Ctx) error {
	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	fileID := c.Params("fileID")
	node, err := h.vfs.FindNodeByPublicID(c.Context(), h.db, fileID, scope)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	c.Locals("file", node)

	return c.Next()
}

// fetchFile returns file metadata
// @Summary Get file info
// @Description Retrieve metadata for a specific file
// @Tags files
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param fileID path string true "File ID"
// @Success 200 {object} FileInfo "File metadata"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "File not found"
// @Router /accounts/{accountID}/files/{fileID} [get]
func (h *HTTPHandler) fetchFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)
	i := FileInfo{
		Kind:      DirItemKindFile,
		ID:        node.PublicID,
		Name:      node.Name,
		Size:      node.Size,
		MimeType:  node.MimeType,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	}
	return c.JSON(i)
}

// downloadFile streams file content
// @Summary Download file
// @Description Download the file content. May redirect to a signed URL for external storage.
// @Tags files
// @Produce application/octet-stream
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param fileID path string true "File ID"
// @Success 200 {file} binary "File content stream"
// @Success 307 {string} string "Redirect to download URL"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "File not found"
// @Router /accounts/{accountID}/files/{fileID}/content [get]
func (h *HTTPHandler) downloadFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)
	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	content, err := h.vfs.ReadFile(c.Context(), h.db, node, scope)
	if err != nil {
		if errors.Is(err, virtualfs.ErrAccessDenied) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if errors.Is(err, virtualfs.ErrUnsupportedOperation) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	if content.URL != "" {
		return c.Redirect(content.URL, fiber.StatusTemporaryRedirect)
	}

	if content.Reader != nil {
		if node.MimeType != "" {
			c.Set("Content-Type", node.MimeType)
		}
		if content.Size > 0 {
			return c.SendStream(content.Reader, int(content.Size))
		}
		return c.SendStream(content.Reader)
	}

	return httperr.Internal(errors.New("vfs returned neither a reader nor a URL"))
}

// patchFile updates file properties
// @Summary Update file
// @Description Update file properties such as name (rename)
// @Tags files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param fileID path string true "File ID"
// @Param request body patchFileRequest true "File update"
// @Success 200 {object} FileInfo "Updated file metadata"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "File not found"
// @Router /accounts/{accountID}/files/{fileID} [patch]
func (h *HTTPHandler) patchFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)
	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	patch := new(patchFileRequest)
	if err := c.BodyParser(patch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	if patch.Name != "" {
		err := h.vfs.RenameNode(c.Context(), tx, node, patch.Name, scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(FileInfo{
		ID:        node.PublicID,
		Name:      node.Name,
		Size:      node.Size,
		MimeType:  node.MimeType,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	})
}

// deleteFile removes a file
// @Summary Delete file
// @Description Delete a file permanently or move it to trash
// @Tags files
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param fileID path string true "File ID"
// @Param trash query bool false "Move to trash instead of permanent delete" default(false)
// @Success 200 {object} FileInfo "Trashed file info (when trash=true)"
// @Success 204 {string} string "Permanently deleted (when trash=false)"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "File not found"
// @Router /accounts/{accountID}/files/{fileID} [delete]
func (h *HTTPHandler) deleteFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)
	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	shouldTrash := c.Query("trash") == "true"
	if shouldTrash {
		deleted, err := h.vfs.SoftDeleteNode(c.Context(), tx, node, scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.JSON(fileInfoFromNode(deleted))
	} else {
		err = h.vfs.PermanentlyDeleteNode(c.Context(), tx, node, scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// deleteFiles removes multiple files
// @Summary Bulk delete files
// @Description Delete multiple files permanently or move them to trash. All items must be files.
// @Tags files
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Param id query string true "Comma-separated list of file IDs to delete" example:"mElnUNCm8F22,kRp2XYTq9A55"
// @Param trash query bool false "Move to trash instead of permanent delete" default(false)
// @Success 200 {array} FileInfo "Trashed files (when trash=true)"
// @Success 204 {string} string "Files deleted"
// @Failure 400 {object} map[string]string "All items must be files"
// @Failure 401 {string} string "Not authenticated"
// @Router /accounts/{accountID}/files [delete]
func (h *HTTPHandler) deleteFiles(c *fiber.Ctx) error {
	scope, ok := scopeFromCtx(c)
	if !ok {
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

	nodes, err := h.vfs.FindNodesByPublicID(c.Context(), tx, ids, scope)
	if err != nil {
		return httperr.Internal(err)
	}

	if len(nodes) == 0 {
		return c.SendStatus(fiber.StatusNoContent)
	}

	if shouldTrash {
		deleted, err := h.vfs.SoftDeleteNodes(c.Context(), tx, nodes, scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		res := make([]FileInfo, 0, len(deleted))
		for _, node := range deleted {
			res = append(res, fileInfoFromNode(node))
		}

		return c.JSON(res)
	} else {
		err = h.vfs.PermanentlyDeleteFiles(c.Context(), tx, nodes, scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrUnsupportedOperation) {
				return httperr.NewHTTPError(fiber.StatusBadRequest, "all items must be files", err)
			}
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}

		err = tx.Commit()
		if err != nil {
			return httperr.Internal(err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

}

// listFileShares returns all shares that include this file
// @Summary List file shares
// @Description Get all share links that include this file
// @Tags files
// @Produce json
// @Param accountID path string true "Account ID" format(uuid)
// @Param fileID path string true "File ID"
// @Success 200 {array} sharing.Share "Array of shares"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "File not found"
// @Security BearerAuth
// @Router /accounts/{accountID}/files/{fileID}/shares [get]
func (h *HTTPHandler) listFileShares(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)

	includesExpired := c.Query("includesExpired") == "true"

	shares, err := h.sharingService.ListShares(c.Context(), h.db, node.AccountID, sharing.ListSharesOptions{
		Items:           []*virtualfs.Node{node},
		IncludesExpired: includesExpired,
	})
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(shares)
}
