package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/get-drexa/drexa/docs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/swaggo/swag"
)

func main() {
	port := flag.Int("port", 8081, "port to listen on")
	apiURL := flag.String("api-url", "http://localhost:8080", "base URL of the API server")
	flag.Parse()

	app := fiber.New(fiber.Config{
		AppName: "Drexa API Documentation",
	})

	app.Use(logger.New())
	app.Use(cors.New())

	// Serve Scalar UI
	app.Get("/", func(c *fiber.Ctx) error {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Drexa API Documentation</title>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%%3E%%3Ctext y='.9em' font-size='90'%%3E📚%%3C/text%%3E%%3C/svg%%3E" />
</head>
<body>
	<script
		id="api-reference"
		data-url="/openapi.json"
		data-configuration='{
			"theme": "kepler",
			"darkMode": true,
			"authentication": {
				"preferredSecurityScheme": "BearerAuth"
			},
			"servers": [
				{
					"url": "%s",
					"description": "API Server"
				}
			]
		}'
	></script>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`, *apiURL)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	// Serve OpenAPI spec
	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		c.Set("Access-Control-Allow-Origin", "*")
		doc, err := swag.ReadDoc()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.SendString(doc)
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	fmt.Fprintf(os.Stderr, "📚 Drexa API Documentation server starting on http://localhost:%d\n", *port)
	fmt.Fprintf(os.Stderr, "   API server configured at: %s\n", *apiURL)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", *port)))
}

