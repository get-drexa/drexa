package httperr

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler is a global error handler for Fiber that logs errors and returns appropriate responses.
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Default status code
	code := fiber.StatusInternalServerError
	message := "Internal"

	// Check if it's our custom HTTPError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		code = httpErr.Code
		message = httpErr.Message

		// Log the error with underlying error details
		if httpErr.Err != nil {
			slog.Error("HTTP error",
				"status", code,
				"message", message,
				"error", httpErr.Err.Error(),
				"path", c.Path(),
				"method", c.Method(),
			)
		} else {
			slog.Warn("HTTP error",
				"status", code,
				"message", message,
				"path", c.Path(),
				"method", c.Method(),
			)
		}
	} else {
		// Check if it's a Fiber error
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			code = fiberErr.Code
			message = fiberErr.Message
		} else {
			// Generic error - log it
			slog.Error("Unhandled error",
				"status", code,
				"error", err.Error(),
				"path", c.Path(),
				"method", c.Method(),
			)
		}
	}

	// Set Content-Type header
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	// Return JSON response
	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

