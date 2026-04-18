package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"

	"jackpotTask/internal/cache"
	"jackpotTask/internal/config"
	httpapi "jackpotTask/internal/http"
	"jackpotTask/internal/http/handlers"
	"jackpotTask/internal/service"
	"jackpotTask/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	mongoStore, err := store.NewMongoStore(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		log.Fatalf("init mongo: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = mongoStore.Close(shutdownCtx)
	}()

	cacheStore := cache.Store(cache.NewMemory(cfg.CacheTTL))
	if cfg.UseRedisCache {
		redisCache := cache.NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
		if err := redisCache.Ping(ctx); err != nil {
			log.Printf("redis unavailable, using in-memory cache: %v", err)
		} else {
			cacheStore = redisCache
			defer func() {
				if err := redisCache.Close(); err != nil {
					log.Printf("redis close error: %v", err)
				}
			}()
			log.Printf("using redis cache at %s", cfg.RedisAddr)
		}
	}

	statsService := service.NewStatsService(mongoStore.Transactions)
	statsHandler := handlers.NewStatsHandler(statsService, cacheStore, validator.New(), cfg.MaxDateRange)
	router := httpapi.NewRouter(statsHandler, cfg.AuthToken, cfg.MongoTimeout)

	httpServer := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("admin stats API listening on %s", cfg.ServerAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

