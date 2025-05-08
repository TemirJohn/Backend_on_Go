package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
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
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}

	// Parse form values manually
	name := c.PostForm("name")
	priceStr := c.PostForm("price")
	description := c.PostForm("description")
	categoryIDStr := c.PostForm("category_id")
	developerIDStr := c.PostForm("developerId")

	if name == "" || priceStr == "" || categoryIDStr == "" || developerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}
	developerID, err := strconv.Atoi(developerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid developer ID"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image is required"})
		return
	}

	// Save file
	filePath := fmt.Sprintf("uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	// Create game model manually
	game := models.Game{
		Name:        name,
		Price:       price,
		Description: description,
		CategoryID:  uint(categoryID),
		DeveloperID: uint(developerID),
		Image:       filePath,
	}

	if err := db.DB.Create(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create game"})
		return
	}

	c.JSON(http.StatusOK, game)
}

func UpdateGame(c *gin.Context) {
	id := c.Param("id")

	var game models.Game
	if err := db.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && (user.Role != "developer" || user.ID != game.DeveloperID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	name := c.PostForm("name")
	price := c.PostForm("price")
	description := c.PostForm("description")

	if name != "" {
		game.Name = name
	}
	if description != "" {
		game.Description = description
	}
	if price != "" {
		if parsedPrice, err := strconv.ParseFloat(price, 64); err == nil {
			game.Price = parsedPrice
		}
	}

	file, err := c.FormFile("image")
	if err == nil {
		imagePath := "uploads/" + file.Filename
		if err := c.SaveUploadedFile(file, imagePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}
		game.Image = imagePath
	}

	if err := db.DB.Save(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game"})
		return
	}

	c.JSON(http.StatusOK, game)
}

func DeleteGame(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Attempting to delete game ID: %s", id)
	var game models.Game
	if err := db.DB.First(&game, id).Error; err != nil {
		log.Printf("Game not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}
	user := c.MustGet("user").(models.User)
	log.Printf("User ID: %d, Role: %s", user.ID, user.Role)
	if user.Role != "admin" && user.Role != "developer" {
		log.Printf("Access denied for role: %s", user.Role)
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}
	if user.Role == "developer" && game.DeveloperID != user.ID {
		log.Printf("Developer %d cannot delete game %s (owned by %d)", user.ID, id, game.DeveloperID)
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own games"})
		return
	}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("game_id = ?", id).Delete(&models.Ownership{}).Error; err != nil {
			log.Printf("Failed to delete ownerships: %v", err)
			return err
		}
		if err := tx.Where("game_id = ?", id).Delete(&models.Review{}).Error; err != nil {
			log.Printf("Failed to delete reviews: %v", err)
			return err
		}
		// Удаление игры
		if err := tx.Delete(&game).Error; err != nil {
			log.Printf("Failed to delete game: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Transaction failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete game: " + err.Error()})
		return
	}
	log.Println("Game deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Game deleted"})
}

func GetGameByID(c *gin.Context) {
	id := c.Param("id")
	var game models.Game

	if err := db.DB.Preload("Category").First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	c.JSON(http.StatusOK, game)
}

func ReturnGame(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	gameID := c.Query("gameId")

	if gameID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gameId is required"})
		return
	}

	var ownership models.Ownership
	if err := db.DB.Where("user_id = ? AND game_id = ?", user.ID, gameID).First(&ownership).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ownership not found"})
		return
	}

	if err := db.DB.Delete(&ownership).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ownership"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Game returned successfully"})
}
