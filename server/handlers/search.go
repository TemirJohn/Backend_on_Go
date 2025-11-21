package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// SearchGames - простой поиск (без concurrency пока)
func SearchGames(c *gin.Context) {
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	start := time.Now()

	var games []models.Game

	// Поиск по названию и описанию
	searchPattern := "%" + query + "%"
	db.DB.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern).
		Preload("Category").
		Limit(50).
		Find(&games)

	duration := time.Since(start)

	c.JSON(http.StatusOK, gin.H{
		"query":       query,
		"results":     games,
		"total_found": len(games),
		"search_time": duration.String(),
	})
}
