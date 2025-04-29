package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

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

	c.JSON(http.StatusOK, review)
}

func GetReviews(c *gin.Context) {
	gameID := c.Query("gameId")
	var reviews []models.Review
	query := db.DB
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	query.Find(&reviews)
	c.JSON(http.StatusOK, reviews)
}

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

	if err := db.DB.Delete(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review deleted"})
}
