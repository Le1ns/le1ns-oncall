package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/local/oncall-plugin/backend/internal/api"
	"github.com/local/oncall-plugin/backend/internal/config"
	"github.com/local/oncall-plugin/backend/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	st, err := store.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("store initialization failed", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(cfg, log, st).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("starting oncall api", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	log.Info("oncall api stopped")
}
