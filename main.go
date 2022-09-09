package main

import (
	"context"
	"log"

	recengine "recengine/internal"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v\n", err)
	}

	engine := recengine.NewEngine()
	if err := engine.LoadDomains(); err != nil {
		log.Printf("Warning: couldn't load domains (first load?): %v\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Error running engine: %v\n", err)
	}

	srv := recengine.NewServer(engine, recengine.MakeServerConfigFromEnv(nil))
	if err := srv.Run(); err != nil {
		log.Fatalf("Error running server: %v\n", err)
	}
}
