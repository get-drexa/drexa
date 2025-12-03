package catalog

import (
	"errors"
	"fmt"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

type FileInfo struct {
	Kind      string     `json:"kind"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	MimeType  string     `json:"mimeType"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

func mustCurrentFileNode(c *fiber.Ctx) *virtualfs.Node {
	return c.Locals("file").(*virtualfs.Node)
}

func (h *HTTPHandler) currentFileMiddleware(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	fileID := c.Params("fileID")
	node, err := h.vfs.FindNodeByPublicID(c.Context(), h.db, account.ID, fileID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	c.Locals("file", node)

	return c.Next()
}

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

func (h *HTTPHandler) downloadFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)

	content, err := h.vfs.ReadFile(c.Context(), h.db, node)
	if err != nil {
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

func (h *HTTPHandler) patchFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)

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

	fmt.Printf("node deleted at: %v\n", node.DeletedAt)

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

func (h *HTTPHandler) deleteFile(c *fiber.Ctx) error {
	node := mustCurrentFileNode(c)

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	shouldTrash := c.Query("trash") == "true"
	if shouldTrash {
		err = h.vfs.SoftDeleteNode(c.Context(), tx, node)
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
