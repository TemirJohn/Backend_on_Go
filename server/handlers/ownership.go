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

	db.DB.Create(&ownership)
	c.JSON(http.StatusOK, gin.H{"message": "Game purchased"})
}

func GetLibrary(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	var ownerships []models.Ownership
	db.DB.Where("user_id = ?", user.ID).Find(&ownerships)

	var games []models.Game
	for _, o := range ownerships {
		var game models.Game
		db.DB.First(&game, o.GameID)
		games = append(games, game)
	}

	c.JSON(http.StatusOK, games)
}
