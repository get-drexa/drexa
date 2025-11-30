package upload

import (
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type createUploadRequest struct {
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
}

type updateUploadRequest struct {
	Status Status `json:"status"`
}

type HTTPHandler struct {
	service *Service
	db      *bun.DB
}

func NewHTTPHandler(s *Service, db *bun.DB) *HTTPHandler {
	return &HTTPHandler{service: s, db: db}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	upload := api.Group("/uploads")

	upload.Post("/", h.Create)
	upload.Put("/:uploadID/content", h.ReceiveContent)
	upload.Patch("/:uploadID", h.Update)
}

func (h *HTTPHandler) Create(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	req := new(createUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	upload, err := h.service.CreateUpload(c.Context(), h.db, account.ID, CreateUploadOptions{
		ParentID: req.ParentID,
		Name:     req.Name,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(upload)
}

func (h *HTTPHandler) ReceiveContent(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	uploadID := c.Params("uploadID")

	err := h.service.ReceiveUpload(c.Context(), h.db, account.ID, uploadID, c.Request().BodyStream())
	defer c.Request().CloseBodyStream()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) Update(c *fiber.Ctx) error {
	account := account.CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	req := new(updateUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Status == StatusCompleted {
		upload, err := h.service.CompleteUpload(c.Context(), h.db, account.ID, c.Params("uploadID"))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
		}
		return c.JSON(upload)
	}

	return c.SendStatus(fiber.StatusBadRequest)
}
