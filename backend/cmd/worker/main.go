package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	w := worker.New(store)
	go w.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	w.Stop()
}
