package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/akksell/rbn/internal/bill"
	"github.com/akksell/rbn/internal/config"
	"github.com/akksell/rbn/internal/gmail"
	"github.com/akksell/rbn/internal/notify"
	"github.com/akksell/rbn/internal/server"
	"github.com/akksell/rbn/internal/store"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	fsClient, err := firestore.NewClient(ctx, cfg.FirestoreProjectID)
	if err != nil {
		log.Fatalf("firestore: %v", err)
	}
	defer fsClient.Close()

	st := store.New(fsClient, cfg)

	gmailClient, err := gmail.NewClient(ctx, cfg.GmailInboxUser)
	if err != nil {
		log.Fatalf("gmail: %v", err)
	}

	extractor := bill.DefaultExtractor()
	sender := notify.NewSender(cfg, gmailClient)

	srv, err := server.New(cfg, st, gmailClient, extractor, sender)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)

	httpServer := &http.Server{Addr: addr, Handler: srv}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
