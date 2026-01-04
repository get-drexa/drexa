package upload

import (
	"errors"
	"fmt"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

// createUploadRequest represents a new upload session request
// @Description Request to initiate a file upload
type createUploadRequest struct {
	// ID of the parent directory to upload into
	ParentID string `json:"parentId" example:"kRp2XYTq9A55"`
	// Name of the file being uploaded
	Name string `json:"name" example:"document.pdf"`
}

// updateUploadRequest represents an upload status update
// @Description Request to update upload status (e.g., mark as completed)
type updateUploadRequest struct {
	// New status for the upload
	Status Status `json:"status" example:"completed" enums:"completed"`
}

type HTTPHandler struct {
	service *Service
	db      *bun.DB
}

func NewHTTPHandler(s *Service, db *bun.DB) *HTTPHandler {
	return &HTTPHandler{service: s, db: db}
}

func (h *HTTPHandler) RegisterRoutes(api *virtualfs.ScopedRouter) {
	upload := api.Group("/uploads")

	upload.Post("/", h.Create)
	upload.Put("/:uploadID/content", h.ReceiveContent)
	upload.Patch("/:uploadID", h.Update)
}

// Create initiates a new file upload session
// @Summary Create upload session
// @Description Start a new file upload session. Returns an upload URL to PUT file content to.
// @Tags uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param request body createUploadRequest true "Upload details"
// @Success 200 {object} Upload "Upload session created"
// @Failure 400 {object} map[string]string "Parent is not a directory"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Parent directory not found"
// @Failure 409 {object} map[string]string "File with this name already exists"
// @Router /drives/{driveID}/uploads [post]
func (h *HTTPHandler) Create(c *fiber.Ctx) error {
	scopeAny := reqctx.VFSAccessScope(c)
	scope, ok := scopeAny.(*virtualfs.Scope)
	if !ok || scope == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	req := new(createUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	upload, err := h.service.CreateUpload(c.Context(), h.db, CreateUploadOptions{
		ParentID: req.ParentID,
		Name:     req.Name,
	}, scope)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if errors.Is(err, virtualfs.ErrAccessDenied) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if errors.Is(err, ErrNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if errors.Is(err, ErrParentNotDirectory) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Parent is not a directory"})
		}
		if errors.Is(err, ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "A file with this name already exists"})
		}
		return httperr.Internal(err)
	}

	if upload.UploadURL == "" {
		upload.UploadURL = fmt.Sprintf("%s%s/%s/content", c.BaseURL(), c.OriginalURL(), upload.ID)
	}

	return c.JSON(upload)
}

// ReceiveContent receives the file content for an upload
// @Summary Upload file content
// @Description Stream file content to complete an upload. Send raw binary data in the request body.
// @Tags uploads
// @Accept application/octet-stream
// @Security BearerAuth
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param uploadID path string true "Upload session ID"
// @Param file body []byte true "File content (binary)"
// @Success 204 {string} string "Content received successfully"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Upload session not found"
// @Router /drives/{driveID}/uploads/{uploadID}/content [put]
func (h *HTTPHandler) ReceiveContent(c *fiber.Ctx) error {
	scopeAny := reqctx.VFSAccessScope(c)
	scope, ok := scopeAny.(*virtualfs.Scope)
	if !ok || scope == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	uploadID := c.Params("uploadID")

	err := h.service.ReceiveUpload(c.Context(), h.db, uploadID, c.Context().RequestBodyStream(), scope)
	defer c.Context().Request.CloseBodyStream()
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if errors.Is(err, virtualfs.ErrAccessDenied) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if errors.Is(err, ErrNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Update updates the upload status
// @Summary Complete upload
// @Description Mark an upload as completed after content has been uploaded. This finalizes the file in the filesystem.
// @Tags uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param uploadID path string true "Upload session ID"
// @Param request body updateUploadRequest true "Status update"
// @Success 200 {object} Upload "Upload completed"
// @Failure 400 {object} map[string]string "Content not uploaded yet or invalid status"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Upload session not found"
// @Router /drives/{driveID}/uploads/{uploadID} [patch]
func (h *HTTPHandler) Update(c *fiber.Ctx) error {
	scopeAny := reqctx.VFSAccessScope(c)
	scope, ok := scopeAny.(*virtualfs.Scope)
	if !ok || scope == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	req := new(updateUploadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Status == StatusCompleted {
		upload, err := h.service.CompleteUpload(c.Context(), h.db, c.Params("uploadID"), scope)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			if errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			if errors.Is(err, ErrNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			if errors.Is(err, ErrContentNotUploaded) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content has not been uploaded"})
			}
			return httperr.Internal(err)
		}
		return c.JSON(upload)
	}

	return c.SendStatus(fiber.StatusBadRequest)
}
