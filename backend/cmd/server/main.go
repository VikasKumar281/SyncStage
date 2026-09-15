
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

	"github.com/vikas/media-sequencer/backend/internal/api"
	"github.com/vikas/media-sequencer/backend/internal/config"
	"github.com/vikas/media-sequencer/backend/internal/models"
	"github.com/vikas/media-sequencer/backend/internal/seed"
	"github.com/vikas/media-sequencer/backend/internal/storage/jsonstore"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[sequencer] ")

	cfg := config.Load()


	store, err := jsonstore.Open(cfg.DataPath, func() *models.State {
		log.Printf("no state file at %s, writing seed data", cfg.DataPath)
		return seed.Default(time.Now())
	})
	if err != nil {
		log.Fatalf("storage: %v", err)
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
		log.Printf("listening on :%s (cycle=%dms, data=%s)", cfg.Port, cfg.CycleMs, cfg.DataPath)
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
