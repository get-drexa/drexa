package main

import (
	"fmt"
	"log"

	"github.com/get-drexa/drexa/internal/drexa"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	config, err := drexa.ServerConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	server := drexa.NewServer(*config)

	log.Fatal(server.Listen(fmt.Sprintf(":%d", config.Port)))
}
