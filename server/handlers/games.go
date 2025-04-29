package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"fmt"
	"github.com/gin-gonic/gin"
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

func GetGameByID(c *gin.Context) {
	id := c.Param("id")
	var game models.Game

	if err := db.DB.Preload("Category").First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	c.JSON(http.StatusOK, game)
}
