package handlers

import (
	"awesomeProject/concurrent"
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// GetDashboardStatistics - получение статистики с concurrency
func GetDashboardStatistics(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	// Параллельный расчет статистики
	start := time.Now()
	stats, err := concurrent.CalculateDashboardStats()
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics":       stats,
		"calculation_time": duration.String(),
	})
}

// SearchGamesAdvanced - продвинутый поиск с concurrency
func SearchGamesAdvanced(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Параллельный поиск
	result, err := concurrent.ParallelSearch(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":       query,
		"results":     result.Games,
		"total_found": result.TotalFound,
		"search_time": result.SearchTime.String(),
	})
}

// ValidateAllGames - валидация всех игр в базе
func ValidateAllGames(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	// Получаем все игры
	var games []models.Game
	db.DB.Find(&games)

	if len(games) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No games to validate"})
		return
	}

	// Параллельная валидация
	start := time.Now()
	results, err := concurrent.ProcessBulkGames(games, "validate", 0.0, 10)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Группируем результаты
	var validGames []uint
	var invalidGames []struct {
		GameID uint
		Error  string
	}

	for _, result := range results {
		if result.Success {
			validGames = append(validGames, result.GameID)
		} else {
			invalidGames = append(invalidGames, struct {
				GameID uint
				Error  string
			}{
				GameID: result.GameID,
				Error:  result.Error.Error(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_games":     len(games),
		"valid_games":     len(validGames),
		"invalid_games":   len(invalidGames),
		"invalid_details": invalidGames,
		"validation_time": duration.String(),
	})
}
