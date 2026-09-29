package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/asceded/url-shortener/internal/config"
	"github.com/asceded/url-shortener/internal/handler"
	"github.com/asceded/url-shortener/internal/repository"
	"github.com/asceded/url-shortener/internal/service"
	"github.com/asceded/url-shortener/internal/worker"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := repository.NewPostgresPool(ctx, cfg.DBDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		return err
	}

	linkRepo := repository.NewLinkRepository(pool)
	clickRepo := repository.NewClickRepository(pool)
	cache := repository.NewLinkCache(redisClient)

	clickWorker := worker.NewClickWorker(clickRepo, 10000, 100, 2*time.Second, log)
	clickWorker.Start(ctx)

	svc := service.NewLinkService(linkRepo, clickRepo, cache, clickWorker, cfg.BaseURL, log)
	h := handler.NewLinkHandler(svc)

	router := gin.New()
	router.Use(gin.Recovery())
	router.POST("/api/links", h.Create)
	router.GET("/api/links/:code/stats", h.Stats)
	router.DELETE("/api/links/:code", h.Delete)
	router.GET("/:code", h.Redirect)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", "error", err)
	}

	log.Info("application stopped")
	return nil
}
