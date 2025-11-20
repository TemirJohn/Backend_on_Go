package main

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/handlers"
	"awesomeProject/middleware"
	"awesomeProject/models"
	"awesomeProject/monitoring"
	"awesomeProject/utils"
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize logger FIRST
	utils.InitLogger()
	utils.Log.Info("🚀 Starting application...")

	db.InitDB()
	utils.Log.Info("✅ Database connected and migrated")

	// Initialize Redis Cache
	if err := cache.InitRedis(); err != nil {
		utils.Log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("⚠️  Redis connection failed, running without cache")
	} else {
		utils.Log.Info("✅ Redis cache connected")

		// Optional: Warm up cache with frequently accessed data
		// cache.WarmCache()
	}

	// Ensure Redis closes on shutdown
	defer func() {
		if err := cache.CloseRedis(); err != nil {
			utils.Log.Error("Failed to close Redis connection", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	monitoring.InitMetrics()
	utils.Log.Info("📊 Prometheus metrics initialized")

	// Set to release mode in production
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Request Logging Middleware (FIRST for all requests)
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorLogger())

	// Prometheus Metrics Middleware
	r.Use(monitoring.PrometheusMiddleware())

	// Security Headers Middleware (FIRST!)
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimitInfo())
	r.Use(middleware.RemovePoweredBy())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	r.Static("/uploads", "./uploads")

	// Prometheus metrics endpoint
	r.GET("/metrics", monitoring.PrometheusHandler())

	// Health check endpoint with Redis status
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

	// Cache flush endpoint (admin only, use with caution!)
	r.POST("/cache/flush", handlers.AuthMiddleware(), func(c *gin.Context) {
		user := c.MustGet("user").(models.User)
		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
			return
		}

		if !cache.IsRedisAvailable() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis not available"})
			return
		}

		if err := cache.FlushAll(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		utils.Log.Warn("Cache flushed by admin", map[string]interface{}{
			"user_id": user.ID,
		})

		c.JSON(http.StatusOK, gin.H{"message": "Cache flushed successfully"})
	})

	// CSRF token endpoint (public, no auth required)
	r.GET("/csrf-token", middleware.GetCSRFTokenHandler)

	// Public routes (no auth, no CSRF)
	r.POST("/login", handlers.Login)
	r.POST("/users", handlers.Register)
	r.GET("/games", handlers.GetGames)
	r.GET("/games/:id", handlers.GetGameByID)
	r.GET("/categories", handlers.GetCategories)
	r.GET("/reviews", handlers.GetReviews)

	// Protected routes (auth + CSRF required)
	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware())
	protected.Use(middleware.CSRFProtection())
	{
		protected.POST("/games", handlers.CreateGame)
		protected.PUT("/games/:id", handlers.UpdateGame)
		protected.DELETE("/games/:id", handlers.DeleteGame)
		protected.DELETE("/ownership", handlers.ReturnGame)
		protected.GET("/library", handlers.GetLibrary)
		protected.POST("/ownership", handlers.BuyGame)
		protected.POST("/categories", handlers.CreateCategory)
		protected.PUT("/categories/:id", handlers.UpdateCategory)
		protected.DELETE("/categories/:id", handlers.DeleteCategory)
		protected.GET("/users", handlers.GetUsers)
		protected.DELETE("/users/:id", handlers.DeleteUser)
		protected.PUT("/users/:id", handlers.UpdateUser)
		protected.GET("/users/:id", handlers.GetUserByID)
		protected.POST("/users/:id/ban", handlers.BanUser)
		protected.POST("/users/:id/unban", handlers.UnbanUser)
		protected.POST("/reviews", handlers.CreateReview)
		protected.DELETE("/reviews/:id", handlers.DeleteReview)
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
	}).Info("🎯 Server configuration")

	if useHTTPS && certFile != "" && keyFile != "" {
		// HTTPS Configuration
		log.Println("🔒 Starting server with HTTPS on port", port)
		log.Println("📜 Certificate:", certFile)
		log.Println("🔑 Private Key:", keyFile)
		log.Println("🔐 Security Headers: ENABLED")
		log.Println("🛡️  CSRF Protection: ENABLED")

		// TLS Configuration with secure defaults
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
		// HTTP Configuration
		log.Println("🌐 Starting server with HTTP on port", port)
		log.Println("⚠️  WARNING: Running without HTTPS. Set USE_HTTPS=true for production")
		log.Println("🛡️  CSRF Protection: ENABLED")
		log.Println("🔐 Security Headers: ENABLED")

		if err := r.Run(":" + port); err != nil {
			log.Fatal("❌ Failed to start server:", err)
		}
	}
}
