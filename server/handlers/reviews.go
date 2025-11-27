package handlers

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// CreateReview with cache invalidation
func CreateReview(c *gin.Context) {
	var review models.Review

	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(models.User)
	review.UserID = user.ID

	if err := db.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	// Invalidate reviews cache for this game
	go func(gID uint) {
		if cache.IsRedisAvailable() {
			cache.InvalidateReviews(gID)
			utils.Log.Info(fmt.Sprintf("Reviews cache invalidated for game %d (ASYNC)", gID))
		}
	}(review.GameID)

	c.JSON(http.StatusOK, review)
}

// GetReviews with Redis caching
func GetReviews(c *gin.Context) {
	gameID := c.Query("gameId")

	// If specific game requested, try cache
	if gameID != "" {
		gID, err := strconv.Atoi(gameID)
		if err == nil && cache.IsRedisAvailable() {
			cachedReviews, err := cache.GetReviews(uint(gID))
			if err == nil && cachedReviews != nil {
				utils.Log.Debug(fmt.Sprintf("Cache HIT: reviews for game %s", gameID))
				c.JSON(http.StatusOK, cachedReviews)
				return
			}
			utils.Log.Debug(fmt.Sprintf("Cache MISS: reviews for game %s", gameID))
		}
	}

	// Fetch from database
	var reviews []models.Review
	query := db.DB.Preload("User")
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	if err := query.Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	// Cache if specific game
	if gameID != "" {
		gID, err := strconv.Atoi(gameID)
		if err == nil && cache.IsRedisAvailable() {
			cache.SetReviews(uint(gID), reviews)
		}
	}

	c.JSON(http.StatusOK, reviews)
}

// DeleteReview with cache invalidation
func DeleteReview(c *gin.Context) {
	id := c.Param("id")
	var review models.Review
	if err := db.DB.First(&review, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.ID != review.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or review author can delete"})
		return
	}

	//gameID := review.GameID // Save before deletion

	if err := db.DB.Delete(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete review"})
		return
	}

	// Invalidate reviews cache for this game
	go func(gID uint) {
		if cache.IsRedisAvailable() {
			cache.InvalidateReviews(gID)
			utils.Log.Info(fmt.Sprintf("Reviews cache invalidated for game %d (ASYNC)", gID))
		}
	}(review.GameID)

	c.JSON(http.StatusOK, gin.H{"message": "Review deleted"})
}
