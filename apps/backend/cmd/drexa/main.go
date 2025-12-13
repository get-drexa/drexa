package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/get-drexa/drexa/docs"
	"github.com/get-drexa/drexa/internal/drexa"
)

// @title Drexa API
// @version 1.0
// @description Drexa is a file storage and management API. It provides endpoints for authentication, user management, file uploads, and virtual filesystem operations.

// @contact.name Drexa Support
// @contact.url https://github.com/get-drexa/drexa

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token. Format: "Bearer {token}"

func main() {
	configPath := flag.String("config", "", "path to config file (required)")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		flag.Usage()
		os.Exit(1)
	}

	config, err := drexa.ConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	server, err := drexa.NewServer(*config)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.Start())
}
