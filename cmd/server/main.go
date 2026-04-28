// @title           TripApp API
// @version         1.0
// @description     Multi-person collaborative travel platform (modular monolith)
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization

package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/database"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/logger"
	"github.com/Ans1110/trip-app/pkg/middleware"
	pkgredis "github.com/Ans1110/trip-app/pkg/redis"
	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprint(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(cfg.Server.Mode)
	defer logger.Sync()

	db, err := database.InitDB(cfg.Database)
	if err != nil {
		logger.Fatal("initialize postgres", zap.Error(err))
	}

	rdb, err := pkgredis.InitRedis(cfg.Redis)
	if err != nil {
		logger.Fatal("initialize redis", zap.Error(err))
	}

	if err := database.RunMigrations(db, "./migrations"); err != nil {
		logger.Fatal("run migrations", zap.Error(err))
	}

	bus := event.New(logger)

	privateKey, err := loadPrivateKey(cfg.JWT.PrivateKeyPath)
	if err != nil {
		logger.Fatal("load private key", zap.Error(err))
	}
	publicKey := &privateKey.PublicKey

	// start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Router
	gin.SetMode(cfg.Server.Mode)
	r := setupRouter(cfg, logger, publicKey, rdb)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info("starting server", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
	logger.Info("server exiting")
}

func setupRouter(
	cfg *config.Config,
	logger *zap.Logger,
	publicKey *rsa.PublicKey,
	rdb *redis.Client,
) *gin.Engine {
	r := gin.New()

	if len(cfg.Server.TrustedProxies) > 0 {
		r.SetTrustedProxies(cfg.Server.TrustedProxies)
	} else {
		r.SetTrustedProxies(nil)
	}

	// Global middleware
	r.Use(
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.TraceID(),
	)
	return r
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM data")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
}
