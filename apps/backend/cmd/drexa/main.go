package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

	server, err := drexa.NewServer(*config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server on :%d", config.Server.Port)
	log.Fatal(server.Listen(fmt.Sprintf(":%d", config.Server.Port)))
}
