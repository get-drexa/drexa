package catalog

import (
	"errors"
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

type DirectoryInfo struct {
	Kind      string         `json:"kind"`
	ID        string         `json:"id"`
	Path      virtualfs.Path `json:"path,omitempty"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt *time.Time     `json:"deletedAt,omitempty"`
}

type createDirectoryRequest struct {
	ParentID string `json:"parentID"`
	Name     string `json:"name"`
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

	include := c.Query("include")
	if include == "path" {
		p, err := h.vfs.RealPath(c.Context(), h.db, node)
		if err != nil {
			return httperr.Internal(err)
		}
		i.Path = p
	}

	return c.JSON(i)
}

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
