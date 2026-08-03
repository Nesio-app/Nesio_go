package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/api"
	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/Nesio-app/Nesio_go/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := storage.New(ctx, os.Getenv("DATABASE_URL"), os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	if err := connector.MigrateLegacyCredentials(store); err != nil {
		log.Fatalf("failed to migrate legacy connector credentials: %v", err)
	}
	if err := storage.EnsureQdrantCollection(ctx, os.Getenv("QDRANT_URL"), "item_visual", 16); err != nil {
		log.Printf("warning: ensure qdrant item_visual collection failed: %v", err)
	}

	backgroundWorker := worker.New(store)
	go backgroundWorker.Start()

	server := api.NewServer(store)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		if err := server.Start(":" + port); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	backgroundWorker.Stop()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
