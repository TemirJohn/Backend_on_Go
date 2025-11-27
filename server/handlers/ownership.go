package handlers

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// BuyGame with cache invalidation
func BuyGame(c *gin.Context) {
	var ownership models.Ownership
	if err := c.ShouldBindJSON(&ownership); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(models.User)
	ownership.UserID = user.ID
	ownership.Status = "owned"

	var game models.Game
	if err := db.DB.First(&game, ownership.GameID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	var existing models.Ownership
	if err := db.DB.Where("user_id = ? AND game_id = ?", user.ID, ownership.GameID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Game already owned"})
		return
	}

	if err := db.DB.Create(&ownership).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purchase game"})
		return
	}

	go func(uID uint) {
		if cache.IsRedisAvailable() {
			cache.InvalidateUserLibrary(uID)
			cache.InvalidateDashboardStats()
			utils.Log.Info(fmt.Sprintf("Library cache invalidated for user %d (ASYNC)", uID))
		}
	}(user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Game purchased"})
}

// GetLibrary with Redis caching
func GetLibrary(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	// Try cache first
	if cache.IsRedisAvailable() {
		cachedLibrary, err := cache.GetUserLibrary(user.ID)
		if err == nil && cachedLibrary != nil {
			utils.Log.Debug(fmt.Sprintf("Cache HIT: library for user %d", user.ID))
			c.JSON(http.StatusOK, cachedLibrary)
			return
		}
		utils.Log.Debug(fmt.Sprintf("Cache MISS: library for user %d", user.ID))
	}

	// Fetch from database
	var ownerships []models.Ownership
	if err := db.DB.Where("user_id = ?", user.ID).Preload("Game").Find(&ownerships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch library"})
		return
	}

	var games []models.Game
	for _, o := range ownerships {
		games = append(games, o.Game)
	}

	// Cache the result
	if cache.IsRedisAvailable() {
		cache.SetUserLibrary(user.ID, games)
	}

	c.JSON(http.StatusOK, games)
}
