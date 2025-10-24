package main

import (
	"awesomeProject/db"
	"awesomeProject/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db.InitDB()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
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
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
