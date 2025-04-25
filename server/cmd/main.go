//package main
//
//import (
//	"awesomeProject/db"
//	"awesomeProject/handlers"
//	"github.com/gin-contrib/cors"
//	"github.com/gin-gonic/gin"
//)
//
//func main() {
//
//	db.InitDB()
//	r := gin.Default()
//
//	r.POST("/users", handlers.Register) // Added register route
//
//	r.Use(cors.New(cors.Config{
//		AllowOrigins:     []string{"http://localhost:3000"},
//		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
//		AllowHeaders:     []string{"Content-Type"},
//		AllowCredentials: true,
//	}))
//
//	r.POST("/login", handlers.Login)
//	r.GET("/games", handlers.GetGames)
//	r.POST("/games", handlers.AuthMiddleware(), handlers.CreateGame)
//	r.POST("/ownership", handlers.AuthMiddleware(), handlers.BuyGame)
//	r.GET("/library", handlers.AuthMiddleware(), handlers.GetLibrary)
//	r.GET("/categories", handlers.GetCategories)
//	r.POST("/categories", handlers.AuthMiddleware(), handlers.CreateCategory)
//	r.PUT("/games/:id", handlers.AuthMiddleware(), handlers.UpdateGame)
//	r.DELETE("/games/:id", handlers.AuthMiddleware(), handlers.DeleteGame)
//	r.Run(":8080")
//}

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
	r.Use(cors.Default())

	// Public routes
	r.POST("/login", handlers.Login)
	r.POST("/users", handlers.Register) // Ensure this line is present

	// Protected routes
	protected := r.Group("/").Use(handlers.AuthMiddleware())
	{
		protected.GET("/games", handlers.GetGames)
		protected.POST("/games", handlers.CreateGame)
		protected.GET("/library", handlers.GetLibrary)
		protected.POST("/ownership", handlers.BuyGame)
		// Add other protected routes as needed
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
