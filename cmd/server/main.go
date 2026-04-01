package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Lactoseandtolerance/bubble-bath/internal/auth"
	"github.com/Lactoseandtolerance/bubble-bath/internal/config"
	"github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/handlers"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	ctx := context.Background()

	pool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()

	tokenEnc := crypto.NewTokenEncryptor(cfg.TokenSecretKey)
	colEnc := crypto.NewColumnEncryptor(cfg.ColumnEncryptionKey)
	userStore := store.NewUserStore(pool)
	authSvc := auth.NewService(userStore, tokenEnc, colEnc, cfg.AccessTokenTTLMinutes, cfg.RefreshTokenTTLDays, cfg.BaseTolerance, cfg.ToleranceFloor, cfg.ToleranceCeiling)

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parsing redis URL: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	authHandler := handlers.NewAuthHandler(authSvc)
	verifyHandler := handlers.NewVerifyHandler(tokenEnc, userStore)
	router := handlers.NewRouter(authHandler, verifyHandler, rdb, cfg.MaxLoginAttemptsPerMinute)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("bubble bath listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
