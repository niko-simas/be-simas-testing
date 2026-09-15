package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"simas-backend/internal/auth"
	"simas-backend/internal/config"
	_ "simas-backend/docs"
	"simas-backend/internal/handler"
	"simas-backend/internal/infrastructure/mailer"
	"simas-backend/internal/infrastructure/storage"
	"simas-backend/internal/middleware"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
	"simas-backend/internal/service"
)

// @title SIMAS Backend API
// @version 1.0
// @description Backend API untuk Sistem Informasi Manajemen Siswa
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	// Database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.Username, cfg.DB.Password, cfg.DB.Name)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.Sekolah{},
		&model.SekolahStateLog{},
		&model.Pendidik{},
		&model.PendidikJadwal{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	log.Println("database migrated")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("warning: redis connection failed: %v", err)
	} else {
		log.Println("redis connected")
	}

	// Infrastructure
	smtpMailer := mailer.NewSMTPMailer(&cfg.Mail)

	var storageClient storage.Storage
	if cfg.S3.AccessKey != "" && cfg.S3.SecretKey != "" {
		s3Storage, err := storage.NewS3Storage(storage.S3Config{
			Bucket:    cfg.S3.Bucket,
			Region:    cfg.S3.Region,
			AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey,
			Endpoint:  cfg.S3.Endpoint,
		})
		if err != nil {
			log.Printf("warning: failed to init S3 storage: %v", err)
		} else {
			storageClient = s3Storage
			log.Println("S3 storage initialized")
		}
	}

	// Auth
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, redisClient)
	googleOAuth := auth.NewGoogleOAuth(&cfg.OAuth)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	rtRepo := repository.NewRefreshTokenRepository(db)
	sekolahRepo := repository.NewSekolahRepository(db)
	sekolahLogRepo := repository.NewSekolahStateLogRepository(db)
	pendidikRepo := repository.NewPendidikRepository(db)
	jadwalRepo := repository.NewPendidikJadwalRepository(db)

	// Services
	otpService := service.NewOTPService(redisClient)
	mobileAuthService := service.NewMobileAuthService(userRepo, rtRepo, otpService, jwtManager, googleOAuth, smtpMailer, redisClient)
	internalAuthService := service.NewInternalAuthService(userRepo, rtRepo, jwtManager)
	sekolahService := service.NewSekolahService(sekolahRepo, sekolahLogRepo, userRepo, jwtManager, smtpMailer, storageClient)
	pendidikService := service.NewPendidikService(pendidikRepo, jadwalRepo, userRepo, sekolahRepo, smtpMailer)

	// Handlers
	authHandler := handler.NewAuthHandler(userRepo, jwtManager, googleOAuth, mobileAuthService, cfg.OAuth.GoogleClientID)
	mobileAuthHandler := handler.NewMobileAuthHandler(mobileAuthService)
	internalAuthHandler := handler.NewInternalAuthHandler(internalAuthService)
	sekolahHandler := handler.NewSekolahHandler(sekolahService)
	pendidikHandler := handler.NewPendidikHandler(pendidikService, sekolahService)

	// Middlewares
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, userRepo)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient)

	// Router
	r := gin.Default()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	api := r.Group("/api/v1")
	{
		// Auth Mobile
		authMobile := api.Group("/auth/mobile")
		{
			authMobile.POST("/register", rateLimitMiddleware.Limit("register"), mobileAuthHandler.Register)
			authMobile.POST("/verify-otp", rateLimitMiddleware.Limit("verify_otp"), mobileAuthHandler.VerifyOTP)
			authMobile.POST("/resend-otp", rateLimitMiddleware.Limit("resend_otp"), mobileAuthHandler.ResendOTP)
			authMobile.POST("/login", rateLimitMiddleware.Limit("login"), mobileAuthHandler.Login)
			authMobile.POST("/oauth", rateLimitMiddleware.Limit("oauth"), mobileAuthHandler.OAuth)
			authMobile.POST("/oauth/complete", rateLimitMiddleware.Limit("oauth"), mobileAuthHandler.CompleteOAuth)
		}

		// Auth Refresh
		api.POST("/auth/refresh", rateLimitMiddleware.Limit("refresh"), authHandler.Refresh)

		// Auth Internal
		authInternal := api.Group("/auth/internal")
		{
			authInternal.POST("/pusat/login", internalAuthHandler.PusatLogin)
			authInternal.POST("/login", internalAuthHandler.InternalLogin)
			authInternal.PUT("/update-password", internalAuthHandler.ForceUpdatePassword)
		}

		// Auth legacy
		authLegacy := api.Group("/auth")
		{
			authLegacy.POST("/google", rateLimitMiddleware.Limit("oauth"), authHandler.GoogleAuth)
			authLegacy.POST("/oauth/complete", authHandler.CompleteOAuth)
			authLegacy.POST("/logout", authHandler.Logout)
			authLegacy.GET("/config", authHandler.Config)
		}

		// Sekolah public
		api.GET("/sekolah/jenjang", sekolahHandler.ListJenjang)
		api.GET("/sekolah/tracking/:tracking_code", sekolahHandler.Progress)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())
	{
		// Auth
		protected.GET("/auth/me", authHandler.Me)

		// Mobile status
		protected.GET("/mobile/status-verifikasi", mobileAuthHandler.StatusVerifikasi)

		// Sekolah - Admin Sekolah
		sekolahAdmin := protected.Group("/sekolah")
		sekolahAdmin.Use(authMiddleware.RequireRole("admin_sekolah"))
		{
			sekolahAdmin.POST("", sekolahHandler.PengajuanBaru)
			sekolahAdmin.POST("/:id/dokumen", sekolahHandler.UploadDokumen)
			sekolahAdmin.GET("/saya", sekolahHandler.MySekolah)
			sekolahAdmin.PUT("/saya", sekolahHandler.UpdateMySekolah)
			sekolahAdmin.POST("/saya/pengajuan-ulang", sekolahHandler.PengajuanUlang)

			// Pendidik management
			sekolahAdmin.POST("/saya/pendidik", pendidikHandler.Tambah)
			sekolahAdmin.GET("/saya/pendidik", pendidikHandler.List)
			sekolahAdmin.GET("/saya/pendidik/:id", pendidikHandler.Detail)
			sekolahAdmin.PUT("/saya/pendidik/:id", pendidikHandler.Update)
			sekolahAdmin.PUT("/saya/pendidik/:id/status", pendidikHandler.UpdateStatus)

			// Jadwal
			sekolahAdmin.POST("/saya/pendidik/:id/jadwal", pendidikHandler.TambahJadwal)
			sekolahAdmin.GET("/saya/pendidik/:id/jadwal", pendidikHandler.ListJadwal)
			sekolahAdmin.PUT("/saya/pendidik/:id/jadwal/:jadwal_id", pendidikHandler.UpdateJadwal)
			sekolahAdmin.DELETE("/saya/pendidik/:id/jadwal/:jadwal_id", pendidikHandler.HapusJadwal)
		}

		// Sekolah - Super Admin
		sekolahSuperAdmin := protected.Group("/admin/sekolah")
		sekolahSuperAdmin.Use(authMiddleware.RequireRole("super_admin"))
		{
			sekolahSuperAdmin.GET("", sekolahHandler.List)
			sekolahSuperAdmin.GET("/:id", sekolahHandler.Detail)
			sekolahSuperAdmin.POST("/:id/verifikasi", sekolahHandler.Verifikasi)
		}

		// Pendidik - Tenaga Pendidik
		pendidikRoutes := protected.Group("/pendidik")
		pendidikRoutes.Use(authMiddleware.RequireRole("tenaga_pendidik"))
		{
			pendidikRoutes.GET("/saya/jadwal/harian", pendidikHandler.MyJadwalHarian)
			pendidikRoutes.GET("/saya/jadwal/minggu-ini", pendidikHandler.MyJadwalMingguIni)
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}
