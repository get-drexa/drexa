package main

import (
	"log"

	"github.com/get-drexa/drexa/internal/database"
)

func main() {
	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
}
