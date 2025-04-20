package main

import (
	"awesomeProject/db"
	"awesomeProject/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()

	r := gin.Default()
	r.POST("/login", handlers.Login)
	r.GET("/games", handlers.GetGames)
	r.POST("/games", handlers.AuthMiddleware(), handlers.CreateGame)
	r.POST("/ownership", handlers.AuthMiddleware(), handlers.BuyGame)
	r.GET("/library", handlers.AuthMiddleware(), handlers.GetLibrary)
	r.GET("/categories", handlers.GetCategories)
	r.POST("/categories", handlers.AuthMiddleware(), handlers.CreateCategory)

	r.Run(":8080")
}
