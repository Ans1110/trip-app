// @title           TripCraft API
// @version         1.0
// @description     Backend for TripCraft — a multi-person collaborative trip planner. Modular monolith (auth, friend, trip/room) on Postgres + Redis. JWT auth via httpOnly cookies + CSRF. Members collaborate inside rooms joined via room code (QR or raw). Trip status is derived from end_date — `completed` is auto-only and cannot be set manually.
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https
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
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/Ans1110/trip-app/docs"
	"github.com/Ans1110/trip-app/internal/album"
	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/calendar"
	"github.com/Ans1110/trip-app/internal/chat"
	"github.com/Ans1110/trip-app/internal/finance"
	"github.com/Ans1110/trip-app/internal/friend"
	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/internal/realtime"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/Ans1110/trip-app/internal/vote"
	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/database"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/logger"
	"github.com/Ans1110/trip-app/pkg/mail"
	"github.com/Ans1110/trip-app/pkg/middleware"
	pkgredis "github.com/Ans1110/trip-app/pkg/redis"
	"github.com/Ans1110/trip-app/pkg/ticket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = resolveConfigPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(cfg.Server.Mode)
	defer logger.Sync()

	// Postgres + Redis open independent connections, so fan them out in parallel
	var (
		db  *gorm.DB
		rdb *redis.Client
	)
	{
		g, _ := errgroup.WithContext(context.Background())
		g.Go(func() error {
			d, err := database.InitDB(cfg.Database)
			if err != nil {
				return fmt.Errorf("initialize postgres: %w", err)
			}
			db = d
			return nil
		})
		g.Go(func() error {
			r, err := pkgredis.InitRedis(cfg.Redis)
			if err != nil {
				return fmt.Errorf("initialize redis: %w", err)
			}
			rdb = r
			return nil
		})
		if err := g.Wait(); err != nil {
			logger.Fatal("initialize infra", zap.Error(err))
		}
	}

	if err := database.RunMigrations(db, resolveMigrationsPath()); err != nil {
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

	// Auth wiring
	mailer := mail.New(cfg.Notification, logger)
	auditRepo := audit.NewRepository(db)
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(auth.ServiceConfig{
		Repo:       authRepo,
		Logger:     logger,
		PrivateKey: privateKey,
		JWT:        cfg.JWT,
		Security:   cfg.Security,
		Mailer:     mailer,
		Redis:      rdb,
		Audit:      auditRepo,
		Bus:        bus,
	})
	authHandler := auth.NewHandler(authSvc, logger, auth.CookieConfig{
		MaxAge:   cfg.JWT.RefreshTokenTTL,
		Secure:   cfg.Server.Mode == gin.ReleaseMode,
		SameSite: http.SameSiteLaxMode,
	})

	// Friend wiring
	friendRepo := friend.NewRepository(db)
	friendSvc := friend.NewService(friend.ServiceConfig{
		Repo:   friendRepo,
		Logger: logger,
		Audit:  auditRepo,
		Bus:    bus,
	})
	friendHandler := friend.NewHandler(friendSvc, logger)

	// Calendar wiring
	calendarRepo := calendar.NewRepository(db)
	calendarSync := calendar.NewTripSync(logger)

	// Trip / Room wiring
	tripRepo := trip.NewRepository(db)
	tripSvc := trip.NewService(trip.ServiceConfig{
		Repo:   tripRepo,
		Logger: logger,
		Audit:  auditRepo,
		Bus:    bus,
		Cal:    calendarSync,
	})
	tripHandler := trip.NewHandler(tripSvc, logger)

	calendarSvc := calendar.NewService(calendar.ServiceConfig{
		Repo:       calendarRepo,
		Friends:    friendRepo,
		RoomMember: tripRepo,
		Logger:     logger,
		Audit:      auditRepo,
		Bus:        bus,
	})
	calendarHandler := calendar.NewHandler(calendarSvc, logger)

	// Realtime wiring: in-process Hub + Pub/Sub bridge for cross-instance
	// fanout + trip.events Stream for downstream consumers (audit/recap/etc.).
	rtHub := realtime.NewHub(logger)
	rtSvc := realtime.NewService(realtime.ServiceConfig{
		TripRepo: tripRepo,
		Hub:      rtHub,
		Redis:    rdb,
		Logger:   logger,
	})

	rtTickets := ticket.NewStore(ticket.Config{
		Redis:  rdb,
		Prefix: "realtime:ticket:",
		TTL:    60 * time.Second,
	})
	rtHandler := realtime.NewHandler(rtSvc, rtHub, logger, realtime.HandlerConfig{
		AllowedOrigins: cfg.Server.AllowedOrigins,
		Tickets:        rtTickets,
	})
	go rtSvc.Start(ctx)

	// Outbox dispatcher
	rtDispatcher := realtime.NewOutboxDispatcher(realtime.OutboxDispatcherConfig{
		Repo:   tripRepo,
		Redis:  rdb,
		Logger: logger,
	})
	go rtDispatcher.Start(ctx)

	// Vote wiring — piggy-backs on the trip realtime hub for WS fanout so
	// members get poll updates on the same socket that carries itinerary/
	// todo edits.
	voteRepo := vote.NewRepository(db)
	voteSvc := vote.NewService(vote.ServiceConfig{
		Repo:        voteRepo,
		TripAuth:    tripRepo,
		Broadcaster: rtSvc,
		Logger:      logger,
	})
	voteHandler := vote.NewHandler(voteSvc, logger)
	voteScheduler := vote.NewScheduler(vote.SchedulerConfig{
		Service: voteSvc,
		Logger:  logger,
	})
	go voteScheduler.Start(ctx)

	mediaStorage, err := media.NewStorage(cfg.Media)
	if err != nil {
		logger.Fatal("initialize media storage", zap.Error(err))
	}
	// Bound the boot-time bucket check
	ensureCtx, ensureCancel := context.WithTimeout(ctx, 5*time.Second)
	err = mediaStorage.EnsureBucket(ensureCtx)
	ensureCancel()
	if err != nil {
		logger.Fatal("ensure media bucket", zap.Error(err))
	}
	mediaRepo := media.NewRepository(db)
	mediaSvc := media.NewService(media.ServiceConfig{
		Repo:    mediaRepo,
		Storage: mediaStorage,
		Cfg:     cfg.Media,
		Bus:     bus,
		Logger:  logger,
	})
	mediaGC := media.NewGC(media.GCConfig{
		Repo:     mediaRepo,
		Storage:  mediaStorage,
		Interval: cfg.Media.GCInterval,
		Logger:   logger,
	})
	mediaGC.Start(ctx)
	mediaHandler := media.NewHandler(mediaSvc, logger)

	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(finance.ServiceConfig{
		Repo:        financeRepo,
		TripAuth:    tripRepo,
		MediaAuthz:  mediaSvc,
		Broadcaster: rtSvc,
		Logger:      logger,
	})
	financeHandler := finance.NewHandler(financeSvc, logger)

	albumRepo := album.NewRepository(db)
	albumSvc := album.NewService(album.ServiceConfig{
		Repo:        albumRepo,
		TripAuth:    tripRepo,
		Media:       mediaSvc,
		Broadcaster: rtSvc,
		Thumbnailer: album.NewThumbnailGenerator(),
		Exif:        album.NewExifExtractor(),
		Logger:      logger,
	})
	albumHandler := album.NewHandler(albumSvc, logger)

	// Chat wiring: writer (bounded queue → batch INSERT) → service (persist
	// then broadcast via hub + pub/sub).
	chatRepo := chat.NewRepository(db)
	chatHub := chat.NewHub(logger)
	chatWriter := chat.NewWriter(chat.WriterConfig{
		Repo:   chatRepo,
		Logger: logger,
	})
	go chatWriter.Start(ctx)
	chatSvc := chat.NewService(chat.ServiceConfig{
		Repo:       chatRepo,
		TripRepo:   tripRepo,
		MediaAuthz: mediaSvc,
		Hub:        chatHub,
		Redis:      rdb,
		Writer:     chatWriter,
		Logger:     logger,
	})
	go chatSvc.Start(ctx)
	chatTickets := ticket.NewStore(ticket.Config{
		Redis:  rdb,
		Prefix: "chat:ticket:",
		TTL:    60 * time.Second,
	})
	chatHandler := chat.NewHandler(chatSvc, chatHub, logger, chat.HandlerConfig{
		AllowedOrigins: cfg.Server.AllowedOrigins,
		Tickets:        chatTickets,
	})

	// Router
	gin.SetMode(cfg.Server.Mode)
	r := setupRouter(cfg, logger, publicKey, rdb, rtTickets, chatTickets, authHandler, friendHandler, tripHandler, calendarHandler, rtHandler, chatHandler, mediaHandler, voteHandler, financeHandler, albumHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info("starting server",
			zap.Int("port", cfg.Server.Port),
			zap.String("url", cfg.Server.Url),
			zap.String("swagger doc", cfg.Server.Url+"/swagger/index.html"),
		)
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
	rtTickets *ticket.Store,
	chatTickets *ticket.Store,
	authHandler auth.IHandler,
	friendHandler friend.IHandler,
	tripHandler trip.IHandler,
	calendarHandler calendar.IHandler,
	rtHandler realtime.IHandler,
	chatHandler chat.IHandler,
	mediaHandler media.IHandler,
	voteHandler vote.IHandler,
	financeHandler finance.IHandler,
	albumHandler album.IHandler,
) *gin.Engine {
	r := gin.New()

	if len(cfg.Server.TrustedProxies) > 0 {
		r.SetTrustedProxies(cfg.Server.TrustedProxies)
	} else {
		r.SetTrustedProxies(nil)
	}

	// Global middleware
	r.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.TraceID(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.Server.AllowedOrigins),
	)

	// Infra routes
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	bodyLimitMB := cfg.Server.BodyLimitMB
	if bodyLimitMB <= 0 {
		bodyLimitMB = 10
	}
	reqTimeout := cfg.Server.RequestTimeout
	if reqTimeout <= 0 {
		reqTimeout = 30 * time.Second
	}
	bodyLimitMW := middleware.BodyLimit(int64(bodyLimitMB) * 1024 * 1024)
	timeoutMW := middleware.Timeout(reqTimeout)
	rateLimitMax := 120
	if cfg.Server.Mode != gin.ReleaseMode {
		rateLimitMax = 1000
	}
	rateLimitMW := middleware.RateLimit(rdb, rateLimitMax, time.Minute, logger)
	csrfMW := middleware.CSRFProtect(rdb)
	secureCookie := cfg.Server.Mode == gin.ReleaseMode

	r.GET("/api/v1/csrf", middleware.CSRFTokenHandler(rdb, secureCookie))

	api := r.Group("/api/v1")
	// Public routes (no JWT required)
	public := api.Group("/")
	public.Use(bodyLimitMW, timeoutMW, rateLimitMW)

	// Protected routes (JWT required)
	jwtMW := middleware.JWTAuth(publicKey, rdb)
	protected := api.Group("/")
	protected.Use(jwtMW, bodyLimitMW, timeoutMW, rateLimitMW, csrfMW)

	authHandler.RegisterRoutes(public, protected)
	friendHandler.RegisterRoutes(protected)
	tripHandler.RegisterRoutes(protected)
	calendarHandler.RegisterRoutes(protected)
	mediaHandler.RegisterRoutes(protected)
	voteHandler.RegisterRoutes(protected)
	financeHandler.RegisterRoutes(protected)
	albumHandler.RegisterRoutes(public, protected)

	ws := r.Group("/")
	ws.Use(middleware.TicketAuth(rtTickets, logger), rateLimitMW)
	rtHandler.RegisterRoutes(protected, ws)

	chatWS := r.Group("/")
	chatWS.Use(middleware.TicketAuth(chatTickets, logger), rateLimitMW)
	chatHandler.RegisterRoutes(protected, chatWS)

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

func resolveConfigPath() string {
	candidates := []string{
		"./config/config.yml",              // cwd = services/api/
		"./api/config/config.yml",          // cwd = services/
		"./services/api/config/config.yml", // cwd = repo root
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "./config/config.yml"
}

func resolveMigrationsPath() string {
	candidates := []string{
		"./migrations",
		"./api/migrations",
		"./services/api/migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "./migrations"
}
