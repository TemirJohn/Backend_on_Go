package handlers

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
)

// GetGames with Redis caching
func GetGames(c *gin.Context) {
	categoryID := c.Query("categoryId")

	// Try cache first (only for non-filtered requests or specific category)
	if cache.IsRedisAvailable() {
		var cachedGames interface{}
		var err error

		if categoryID != "" {
			// Try category-specific cache
			catID, _ := strconv.Atoi(categoryID)
			cachedGames, err = cache.GetGamesByCategory(uint(catID))
		} else {
			// Try all games cache
			cachedGames, err = cache.GetGames()
		}

		if err == nil && cachedGames != nil {
			utils.Log.Debug("Cache HIT: games list")
			c.JSON(http.StatusOK, cachedGames)
			return
		}
		utils.Log.Debug("Cache MISS: games list")
	}

	// Fetch from database
	var games []models.Game
	query := db.DB.Preload("Category")

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if err := query.Find(&games).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}

	// Cache the result
	if cache.IsRedisAvailable() {
		if categoryID != "" {
			catID, _ := strconv.Atoi(categoryID)
			cache.SetGamesByCategory(uint(catID), games)
		} else {
			cache.SetGames(games)
		}
	}

	c.JSON(http.StatusOK, games)
}

// GetGameByID with Redis caching
func GetGameByID(c *gin.Context) {
	id := c.Param("id")
	gameID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Try cache first
	if cache.IsRedisAvailable() {
		cachedGame, err := cache.GetGame(uint(gameID))
		if err == nil && cachedGame != nil {
			utils.Log.Debug("Cache HIT: game " + id)
			c.JSON(http.StatusOK, cachedGame)
			return
		}
		utils.Log.Debug("Cache MISS: game " + id)
	}

	// Fetch from database
	var game models.Game
	if err := db.DB.Preload("Category").First(&game, gameID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Cache the result
	if cache.IsRedisAvailable() {
		cache.SetGame(uint(gameID), game)
	}

	c.JSON(http.StatusOK, game)
}

// CreateGame with cache invalidation
func CreateGame(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" && user.Role != "developer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins or developers only"})
		return
	}

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

	filePath := fmt.Sprintf("uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

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

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateGamesList()
		cache.InvalidateDashboardStats()
		utils.Log.Info("Cache invalidated after game creation")
	}

	c.JSON(http.StatusOK, game)
}

// UpdateGame with cache invalidation
func UpdateGame(c *gin.Context) {
	id := c.Param("id")
	gameID, _ := strconv.Atoi(id)

	var game models.Game
	if err := db.DB.First(&game, gameID).Error; err != nil {
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

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateGame(uint(gameID))
		cache.InvalidateGamesList()
		utils.Log.Info(fmt.Sprintf("Cache invalidated for game %d", gameID))
	}

	c.JSON(http.StatusOK, game)
}

// DeleteGame with cache invalidation
func DeleteGame(c *gin.Context) {
	id := c.Param("id")
	gameID, _ := strconv.Atoi(id)

	log.Printf("Attempting to delete game ID: %s", id)

	var game models.Game
	if err := db.DB.First(&game, gameID).Error; err != nil {
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
		if err := tx.Where("game_id = ?", gameID).Delete(&models.Ownership{}).Error; err != nil {
			log.Printf("Failed to delete ownerships: %v", err)
			return err
		}
		if err := tx.Where("game_id = ?", gameID).Delete(&models.Review{}).Error; err != nil {
			log.Printf("Failed to delete reviews: %v", err)
			return err
		}
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

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateGame(uint(gameID))
		cache.InvalidateGamesList()
		cache.InvalidateReviews(uint(gameID))
		cache.InvalidateDashboardStats()
		utils.Log.Info(fmt.Sprintf("All caches invalidated for deleted game %d", gameID))
	}

	log.Println("Game deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Game deleted"})
}

// ReturnGame with cache invalidation
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

	// Invalidate user's library cache
	if cache.IsRedisAvailable() {
		cache.InvalidateUserLibrary(user.ID)
		utils.Log.Info(fmt.Sprintf("Library cache invalidated for user %d", user.ID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Game returned successfully"})
}
