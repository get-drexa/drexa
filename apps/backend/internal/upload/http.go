package upload

import (
	"errors"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/gofiber/fiber/v2"
)

const uploadServiceKey = "uploadService"

type createUploadRequest struct {
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
}

type updateUploadRequest struct {
	Status Status `json:"status"`
}

func RegisterAPIRoutes(api fiber.Router, s *Service) {
	upload := api.Group("/uploads", func(c *fiber.Ctx) error {
		c.Locals(uploadServiceKey, s)
		return c.Next()
	})

	upload.Post("/", createUpload)
	upload.Put("/:uploadID/content", receiveUpload)
	upload.Patch("/:uploadID", updateUpload)
}

func mustUploadService(c *fiber.Ctx) *Service {
	return c.Locals(uploadServiceKey).(*Service)
}

func createUpload(c *fiber.Ctx) error {
	u, err := auth.AuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	s := mustUploadService(c)

	req := new(createUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	upload, err := s.CreateUpload(c.Context(), u.ID, CreateUploadOptions{
		ParentID: req.ParentID,
		Name:     req.Name,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(upload)
}

func receiveUpload(c *fiber.Ctx) error {
	u, err := auth.AuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	s := mustUploadService(c)

	uploadID := c.Params("uploadID")

	err = s.ReceiveUpload(c.Context(), u.ID, uploadID, c.Request().BodyStream())
	defer c.Request().CloseBodyStream()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func updateUpload(c *fiber.Ctx) error {
	u, err := auth.AuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	s := mustUploadService(c)

	req := new(updateUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Status == StatusCompleted {
		upload, err := s.CompleteUpload(c.Context(), u.ID, c.Params("uploadID"))
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
