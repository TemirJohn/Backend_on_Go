package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetGames(c *gin.Context) {
	var games []models.Game
	query := db.DB.Preload("Category")
	if categoryID := c.Query("categoryId"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	query.Find(&games)
	c.JSON(http.StatusOK, games)
}

func CreateGame(c *gin.Context) {

	var game models.Game
	if err := c.ShouldBindJSON(&game); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}

	db.DB.Create(&game)
	c.JSON(http.StatusOK, game)
}

func UpdateGame(c *gin.Context) {
	id := c.Param("id")
	var game models.Game
	if err := db.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}
	if err := c.ShouldBindJSON(&game); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}
	db.DB.Save(&game)
	c.JSON(http.StatusOK, game)
}
func DeleteGame(c *gin.Context) {
	id := c.Param("id")
	var game models.Game
	if err := db.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}
	db.DB.Delete(&game)
	c.JSON(http.StatusOK, gin.H{"message": "Game deleted"})
}
