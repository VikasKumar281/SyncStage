package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VikasKumar281/SyncStage/backend/internal/api"
	"github.com/VikasKumar281/SyncStage/backend/internal/config"
	"github.com/VikasKumar281/SyncStage/backend/internal/models"
	"github.com/VikasKumar281/SyncStage/backend/internal/seed"
	"github.com/VikasKumar281/SyncStage/backend/internal/storage"
	"github.com/VikasKumar281/SyncStage/backend/internal/storage/jsonstore"
	"github.com/VikasKumar281/SyncStage/backend/internal/storage/pgstore"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[sequencer] ")

	cfg := config.Load()

	var store storage.Store

	if cfg.DatabaseURL != "" {
		log.Println("using PostgreSQL storage")

		pgStore, err := pgstore.Open(cfg.DatabaseURL, func() *models.State {
			log.Println("no PostgreSQL state found, writing seed data")
			return seed.Default(time.Now())
		})
		if err != nil {
			log.Fatalf("storage: %v", err)
		}

		store = pgStore
	} else {
		log.Println("DATABASE_URL not set, using JSON file storage")

		jsonStore, err := jsonstore.Open(cfg.DataPath, func() *models.State {
			log.Printf("no state file at %s, writing seed data", cfg.DataPath)
			return seed.Default(time.Now())
		})
		if err != nil {
			log.Fatalf("storage: %v", err)
		}

		store = jsonStore
	}

	defer store.Close()

	server := api.NewServer(cfg, store)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,

		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s (cycle=%dms)", cfg.Port, cfg.CycleMs)

		if cfg.StaticDir != "" {
			log.Printf("serving built frontend from %s", cfg.StaticDir)
		}

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutdown signal received, draining connections")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	log.Println("bye")
}
