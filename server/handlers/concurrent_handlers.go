package handlers

import (
	"awesomeProject/concurrent"
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

// GetGameDetailsAdvanced - расширенная информация об игре с concurrency
// GET /games/:id/details
func GetGameDetailsAdvanced(c *gin.Context) {
	id := c.Param("id")
	gameID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Используем concurrent fetching
	start := time.Now()
	details, err := concurrent.FetchGameWithDetails(uint(gameID))
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game":          details.Game,
		"reviews":       details.Reviews,
		"related_games": details.RelatedGames,
		"statistics":    details.Statistics,
		"fetch_time_ms": duration.Milliseconds(),
	})
}

// BulkUpdateGamePrices - массовое обновление цен с concurrency
// POST /admin/games/bulk-update-prices
func BulkUpdateGamePrices(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var input struct {
		CategoryID *uint   `json:"category_id"`
		Action     string  `json:"action"` // "discount", "validate"
		Percentage float64 `json:"percentage"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем игры для обработки
	var games []models.Game
	query := db.DB.Model(&models.Game{})
	if input.CategoryID != nil {
		query = query.Where("category_id = ?", *input.CategoryID)
	}
	query.Find(&games)

	if len(games) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No games found"})
		return
	}

	// Параллельная обработка
	start := time.Now()
	results, err := concurrent.ProcessBulkGames(games, input.Action, 10)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Подсчет успешных/неуспешных операций
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_games":     len(games),
		"successful":      successful,
		"failed":          failed,
		"results":         results,
		"processing_time": duration.String(),
	})
}

// SendGameReleaseNotifications - отправка уведомлений о новой игре
// POST /games/:id/notify
func SendGameReleaseNotifications(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	id := c.Param("id")
	gameID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Получаем игру
	var game models.Game
	if err := db.DB.First(&game, gameID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Получаем всех активных пользователей
	var users []models.User
	db.DB.Where("is_banned = ?", false).Find(&users)

	// Создаем задания для уведомлений
	notifications := make([]concurrent.NotificationJob, len(users))
	for i, u := range users {
		notifications[i] = concurrent.NotificationJob{
			UserID:  u.ID,
			Message: "New game released: " + game.Name,
			Type:    "push",
		}
	}

	// Отправляем параллельно
	start := time.Now()
	results, err := concurrent.SendBulkNotifications(notifications, 20)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Подсчет результатов
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"game":               game.Name,
		"total_users":        len(users),
		"notifications_sent": successful,
		"failed":             failed,
		"time_taken":         duration.String(),
		"avg_time_per_user":  duration.Milliseconds() / int64(len(users)),
	})
}

// GetDashboardStatistics - получение статистики с concurrency
// GET /admin/dashboard/stats
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
// GET /games/search?q=query
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
// POST /admin/games/validate-all
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
	results, err := concurrent.ProcessBulkGames(games, "validate", 15)
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

// ProcessGameImages - обработка изображений игр (пример)
// POST /admin/games/:id/process-images
func ProcessGameImages(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	id := c.Param("id")
	gameID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// В реальном приложении здесь бы загружались файлы
	// Для демонстрации создадим mock данные
	jobs := []concurrent.ImageProcessingJob{
		{
			GameID:    uint(gameID),
			ImageData: []byte("mock image data 1"),
			Filename:  "image1.jpg",
		},
		{
			GameID:    uint(gameID),
			ImageData: []byte("mock image data 2"),
			Filename:  "image2.jpg",
		},
		{
			GameID:    uint(gameID),
			ImageData: []byte("mock image data 3"),
			Filename:  "image3.jpg",
		},
	}

	start := time.Now()
	results, err := concurrent.ProcessImagesInPipeline(jobs)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id":          gameID,
		"images_processed": len(results),
		"processing_time":  duration.String(),
		"results":          results,
	})
}

// GetUserLibraryWithDetails - библиотека пользователя с детальной информацией
// GET /library/detailed
func GetUserLibraryWithDetails(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	// Получаем игры пользователя
	var ownerships []models.Ownership
	db.DB.Where("user_id = ?", user.ID).Preload("Game").Find(&ownerships)

	if len(ownerships) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"games": []interface{}{},
			"total": 0,
		})
		return
	}

	// Параллельно загружаем детали для каждой игры
	type gameDetail struct {
		Game       models.Game
		Statistics concurrent.GameStatistics
		Error      error
	}

	detailsChan := make(chan gameDetail, len(ownerships))

	for _, ownership := range ownerships {
		go func(gameID uint) {
			details, err := concurrent.FetchGameWithDetails(gameID)
			if err != nil {
				detailsChan <- gameDetail{Error: err}
				return
			}
			detailsChan <- gameDetail{
				Game:       details.Game,
				Statistics: details.Statistics,
			}
		}(ownership.Game.ID)
	}

	// Собираем результаты
	var enrichedGames []gameDetail
	for i := 0; i < len(ownerships); i++ {
		detail := <-detailsChan
		if detail.Error == nil {
			enrichedGames = append(enrichedGames, detail)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"games": enrichedGames,
		"total": len(enrichedGames),
	})
}
