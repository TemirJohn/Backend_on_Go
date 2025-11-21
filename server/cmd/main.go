package main

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/handlers"
	"awesomeProject/middleware"
	"awesomeProject/monitoring"
	"awesomeProject/utils"
	"crypto/tls"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize logger FIRST
	utils.InitLogger()
	utils.Log.Info("🚀 Starting application...")

	// Initialize Database
	db.InitDB()
	utils.Log.Info("✅ Database connected and migrated")

	// Initialize Redis Cache
	if err := cache.InitRedis(); err != nil {
		utils.Log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("⚠️  Redis connection failed, running without cache")
	} else {
		utils.Log.Info("✅ Redis cache connected")
	}

	defer func() {
		if err := cache.CloseRedis(); err != nil {
			utils.Log.Error("Failed to close Redis connection", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Initialize Prometheus metrics
	monitoring.InitMetrics()
	utils.Log.Info("📊 Prometheus metrics initialized")

	// Set to release mode in production
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Request Logging Middleware
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorLogger())

	// Prometheus Metrics Middleware
	r.Use(monitoring.PrometheusMiddleware())

	// Security Headers Middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimitInfo())
	r.Use(middleware.RemovePoweredBy())

	// More lenient global rate limiting for development
	//r.Use(middleware.RateLimitMiddleware(300, time.Minute)) // 300 requests per minute

	// CORS Configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Window"},
		AllowCredentials: true,
	}))

	r.Static("/uploads", "./uploads")

	// Prometheus metrics endpoint
	r.GET("/metrics", monitoring.PrometheusHandler())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		redisStatus := "disconnected"
		if cache.IsRedisAvailable() {
			redisStatus = "connected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
			"cache":   redisStatus,
			"redis":   cache.RedisClient != nil,
		})
	})

	// CSRF token endpoint
	r.GET("/csrf-token", middleware.GetCSRFTokenHandler)

	// ==================== PUBLIC ROUTES ====================
	public := r.Group("/")
	//public.Use(middleware.RateLimitMiddleware(50, time.Minute)) // 50 per minute for public
	{
		public.POST("/login", handlers.Login)
		public.POST("/users", handlers.Register)
		public.GET("/games", handlers.GetGames)
		public.GET("/games/:id", handlers.GetGameByID)
		public.GET("/games/search", handlers.SearchGames) // Search endpoint
		public.GET("/categories", handlers.GetCategories)
		public.GET("/reviews", handlers.GetReviews)
	}

	// ==================== PROTECTED ROUTES ====================
	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware())
	protected.Use(middleware.CSRFProtection())
	{
		// Game management
		protected.POST("/games", handlers.CreateGame)
		protected.PUT("/games/:id", handlers.UpdateGame)
		protected.DELETE("/games/:id", handlers.DeleteGame)

		// Ownership
		protected.DELETE("/ownership", handlers.ReturnGame)
		protected.GET("/library", handlers.GetLibrary)
		protected.POST("/ownership", handlers.BuyGame)

		// Categories
		protected.POST("/categories", handlers.CreateCategory)
		protected.PUT("/categories/:id", handlers.UpdateCategory)
		protected.DELETE("/categories/:id", handlers.DeleteCategory)

		// Users
		protected.GET("/users", handlers.GetUsers)
		protected.DELETE("/users/:id", handlers.DeleteUser)
		protected.PUT("/users/:id", handlers.UpdateUser)
		protected.GET("/users/:id", handlers.GetUserByID)
		protected.POST("/users/:id/ban", handlers.BanUser)
		protected.POST("/users/:id/unban", handlers.UnbanUser)

		// Reviews
		protected.POST("/reviews", handlers.CreateReview)
		protected.DELETE("/reviews/:id", handlers.DeleteReview)
	}

	// ==================== ADMIN ROUTES ====================
	admin := r.Group("/admin")
	admin.Use(handlers.AuthMiddleware())
	admin.Use(middleware.CSRFProtection())
	{
		// Dashboard statistics (simple version)
		admin.GET("/dashboard/stats", handlers.GetDashboardStats)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Check if HTTPS should be enabled
	useHTTPS := os.Getenv("USE_HTTPS") == "true"
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	// Log all enabled features
	utils.Log.WithFields(map[string]interface{}{
		"port":             port,
		"https":            useHTTPS,
		"csrf_protection":  true,
		"security_headers": true,
		"logging":          true,
		"metrics":          true,
		"redis_cache":      cache.IsRedisAvailable(),
		"rate_limiting":    true,
	}).Info("🎯 Server configuration")

	if useHTTPS && certFile != "" && keyFile != "" {
		// HTTPS Configuration
		log.Println("🔒 Starting server with HTTPS on port", port)

		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}

		server := &http.Server{
			Addr:      ":" + port,
			Handler:   r,
			TLSConfig: tlsConfig,
		}

		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Fatal("❌ Failed to start HTTPS server:", err)
		}
	} else {
		log.Println("🌐 Starting server with HTTP on port", port)
		if err := r.Run(":" + port); err != nil {
			log.Fatal("❌ Failed to start server:", err)
		}
	}
}
