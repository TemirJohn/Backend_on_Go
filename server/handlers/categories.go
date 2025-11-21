package handlers

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetCategories with Redis caching
func GetCategories(c *gin.Context) {
	// Try cache first
	if cache.IsRedisAvailable() {
		cachedCategories, err := cache.GetCategories()
		if err == nil && cachedCategories != nil {
			utils.Log.Debug("Cache HIT: categories")
			c.JSON(http.StatusOK, cachedCategories)
			return
		}
		utils.Log.Debug("Cache MISS: categories")
	}

	// Fetch from database
	var categories []models.Category
	if err := db.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	// Cache the result
	if cache.IsRedisAvailable() {
		cache.SetCategories(categories)
	}

	c.JSON(http.StatusOK, categories)
}

// CreateCategory with cache invalidation
func CreateCategory(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	if err := db.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	// Invalidate categories cache
	if cache.IsRedisAvailable() {
		cache.InvalidateCategories()
		utils.Log.Info("Categories cache invalidated after creation")
	}

	c.JSON(http.StatusOK, category)
}

// UpdateCategory with cache invalidation
func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var category models.Category
	if err := db.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	if err := db.DB.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateCategories()
		cache.InvalidateGamesList() // Games include category info
		utils.Log.Info("Categories and games cache invalidated after update")
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory with cache invalidation
func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	var category models.Category
	if err := db.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	if err := db.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateCategories()
		cache.InvalidateGamesList()
		utils.Log.Info("Categories and games cache invalidated after deletion")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted"})
}
