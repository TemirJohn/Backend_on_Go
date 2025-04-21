package main

import (
	"awesomeProject/db"
	"awesomeProject/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	db.InitDB()
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.POST("/login", handlers.Login)
	r.GET("/games", handlers.GetGames)
	r.POST("/games", handlers.AuthMiddleware(), handlers.CreateGame)
	r.POST("/ownership", handlers.AuthMiddleware(), handlers.BuyGame)
	r.GET("/library", handlers.AuthMiddleware(), handlers.GetLibrary)
	r.GET("/categories", handlers.GetCategories)
	r.POST("/categories", handlers.AuthMiddleware(), handlers.CreateCategory)
	r.PUT("/games/:id", handlers.AuthMiddleware(), handlers.UpdateGame)
	r.DELETE("/games/:id", handlers.AuthMiddleware(), handlers.DeleteGame)
	r.Run(":8080")
}
