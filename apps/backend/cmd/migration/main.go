package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drexa"
)

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

	db := database.NewFromPostgres(config.Database.PostgresURL)
	defer db.Close()

	log.Println("running migrations...")
	if err := database.RunMigrations(context.Background(), db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("migrations completed successfully")
}
