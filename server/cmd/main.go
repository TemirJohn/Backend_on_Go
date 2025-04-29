package main

import (
	"awesomeProject/db"
	"awesomeProject/handlers"
	"github.com/gin-contrib/cors" // Added for CORS
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db.InitDB()

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},                   // Разрешить фронтенд
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // Разрешить методы
		AllowHeaders:     []string{"Authorization", "Content-Type"},           // Разрешить нужные заголовки
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true, // если используешь сессии/куки
	}))
	r.Static("/uploads", "./uploads")

	// Public routes
	r.POST("/login", handlers.Login)
	r.POST("/users", handlers.Register) // Ensure this line is present

	// Protected routes
	protected := r.Group("/").Use(handlers.AuthMiddleware())
	{
		protected.GET("/games", handlers.GetGames)
		protected.GET("/games/:id", handlers.GetGameByID)
		protected.POST("/games", handlers.CreateGame)
		protected.PUT("/games/:id", handlers.UpdateGame)
		protected.DELETE("/games/:id", handlers.DeleteGame)
		protected.GET("/library", handlers.GetLibrary)
		protected.POST("/ownership", handlers.BuyGame)
		protected.GET("/categories", handlers.GetCategories)
		protected.POST("/categories", handlers.CreateCategory)
		protected.PUT("/categories/:id", handlers.UpdateCategory)
		protected.DELETE("/categories/:id", handlers.DeleteCategory)
		protected.GET("/users", handlers.GetUsers)
		protected.DELETE("/users/:id", handlers.DeleteUser)
		protected.PUT("/users/:id", handlers.UpdateUser)
		protected.POST("/reviews", handlers.CreateReview)
		protected.GET("/reviews", handlers.GetReviews)
		protected.DELETE("/reviews/:id", handlers.DeleteReview)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
