package main

import (
	"awesomeProject/db"
	"awesomeProject/handlers"
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

	db.InitDB()

	// Set to release mode in production
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	r.Static("/uploads", "./uploads")

	// Public routes
	r.POST("/login", handlers.Login)
	r.POST("/users", handlers.Register)
	r.GET("/games", handlers.GetGames)
	r.GET("/games/:id", handlers.GetGameByID)
	r.GET("/categories", handlers.GetCategories)
	r.GET("/reviews", handlers.GetReviews)

	protected := r.Group("/").Use(handlers.AuthMiddleware())
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

	if useHTTPS && certFile != "" && keyFile != "" {
		// HTTPS Configuration
		log.Println("🔒 Starting server with HTTPS on port", port)
		log.Println("📜 Certificate:", certFile)
		log.Println("🔑 Private Key:", keyFile)

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

		if err := r.Run(":" + port); err != nil {
			log.Fatal("❌ Failed to start server:", err)
		}
	}
}
