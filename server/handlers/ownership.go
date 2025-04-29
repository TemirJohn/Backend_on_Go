package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

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

	db.DB.Create(&ownership)
	c.JSON(http.StatusOK, gin.H{"message": "Game purchased"})
}

func GetLibrary(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	var ownerships []models.Ownership
	db.DB.Where("user_id = ?", user.ID).Preload("Game").Find(&ownerships)

	var games []models.Game
	for _, o := range ownerships {
		games = append(games, o.Game)
	}

	c.JSON(http.StatusOK, games)
}
