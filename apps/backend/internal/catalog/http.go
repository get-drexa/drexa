package catalog

import (
	"errors"
	"fmt"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	vfs *virtualfs.VirtualFS
	db  *bun.DB
}

type patchFileRequest struct {
	Name string `json:"name"`
}

func NewHTTPHandler(vfs *virtualfs.VirtualFS, db *bun.DB) *HTTPHandler {
	return &HTTPHandler{vfs: vfs, db: db}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/files/:fileID")
	g.Use(h.currentFileMiddleware)
	g.Get("/", h.fetchFile)
	g.Get("/content", h.downloadFile)
	g.Patch("/", h.patchFile)
	g.Delete("/", h.deleteFile)
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
		ID:        node.PublicID,
		Name:      node.Name,
		Size:      node.Size,
		MimeType:  node.MimeType,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}
	if node.DeletedAt != nil {
		i.DeletedAt = node.DeletedAt
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
