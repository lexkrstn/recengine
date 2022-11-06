package main

import (
	"context"
	"log"
	"recengine/internal/api/shard"
	"recengine/internal/domain"
	"recengine/internal/infra/delta"
	"recengine/internal/infra/index"

	"github.com/joho/godotenv"
)

func runShard() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deltaStorageFactory := delta.NewStorageFactory()
	indexStorageFactory := index.NewStorageFactory()

	nsService := domain.NewNamespaceService(ctx, deltaStorageFactory, indexStorageFactory)
	if err := nsService.LoadNamespaces(); err != nil {
		log.Printf("Warning: couldn't load domains (first load?): %v\n", err)
	}
	if err := nsService.Start(ctx); err != nil {
		log.Fatalf("Error running namespace service: %v\n", err)
	}

	app := shard.NewApplication(&shard.ApplicationDto{
		Config:    shard.NewConfigFromEnv(nil),
		NsService: nsService,
	})
	if err := app.Run(); err != nil {
		log.Fatalf("Error running shard application: %v\n", err)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v\n", err)
	}
	runShard()
}
