package handlers

import (
	"awesomeProject/cache"
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

// GetUsers - admins only
func GetUsers(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetUserByID with caching
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" && currentUser.ID != uint(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Try cache first
	if cache.IsRedisAvailable() {
		cachedUser, err := cache.GetUser(uint(userID))
		if err == nil && cachedUser != nil {
			utils.Log.Debug(fmt.Sprintf("Cache HIT: user %d", userID))
			c.JSON(http.StatusOK, cachedUser)
			return
		}
		utils.Log.Debug(fmt.Sprintf("Cache MISS: user %d", userID))
	}

	// Fetch from database
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Cache the result
	if cache.IsRedisAvailable() {
		cache.SetUser(uint(userID), user)
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser with cache invalidation
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := strconv.Atoi(id)

	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := db.DB.Delete(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	// Invalidate caches
	if cache.IsRedisAvailable() {
		cache.InvalidateUser(uint(userID))
		cache.InvalidateUserLibrary(uint(userID))
		cache.InvalidateDashboardStats()
		utils.Log.Info(fmt.Sprintf("User %d caches invalidated after deletion", userID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// UpdateUser with cache invalidation
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := strconv.Atoi(id)
	currentUser := c.MustGet("user").(models.User)

	log.Printf("Updating user ID: %s by user ID: %d, role: %s", id, currentUser.ID, currentUser.Role)

	if currentUser.Role != "admin" && currentUser.ID != uint(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, userID).Error; err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	file, err := c.FormFile("avatar")
	if err == nil {
		filename := fmt.Sprintf("uploads/%s", file.Filename)
		log.Printf("Saving avatar to: %s", filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			log.Printf("Failed to save avatar: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
			return
		}
		targetUser.Avatar = filename
		log.Printf("Updated avatar path: %s", targetUser.Avatar)
	}

	var input struct {
		Name     *string `form:"name"`
		Role     *string `form:"role"`
		IsBanned *bool   `form:"isBanned"`
	}
	if err := c.ShouldBind(&input); err != nil {
		log.Printf("Invalid input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != nil {
		targetUser.Name = *input.Name
	}
	if input.Role != nil && currentUser.Role == "admin" {
		targetUser.Role = *input.Role
	}

	if input.IsBanned != nil && currentUser.Role == "admin" {
		targetUser.IsBanned = *input.IsBanned
	}

	if err := db.DB.Save(&targetUser).Error; err != nil {
		log.Printf("Failed to update user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Invalidate user cache
	if cache.IsRedisAvailable() {
		cache.InvalidateUser(uint(userID))
		utils.Log.Info(fmt.Sprintf("User %d cache invalidated after update", userID))
	}

	log.Printf("User updated successfully, avatar: %s", targetUser.Avatar)
	c.JSON(http.StatusOK, targetUser)
}

// BanUser with cache invalidation
func BanUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := strconv.Atoi(id)
	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	targetUser.IsBanned = true
	if err := db.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}

	// Invalidate user cache
	if cache.IsRedisAvailable() {
		cache.InvalidateUser(uint(userID))
		utils.Log.Info(fmt.Sprintf("User %d banned and cache invalidated", userID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "User banned", "user": targetUser})
}

// UnbanUser with cache invalidation
func UnbanUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := strconv.Atoi(id)
	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	targetUser.IsBanned = false
	if err := db.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unban user"})
		return
	}

	// Invalidate user cache
	if cache.IsRedisAvailable() {
		cache.InvalidateUser(uint(userID))
		utils.Log.Info(fmt.Sprintf("User %d unbanned and cache invalidated", userID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unbanned", "user": targetUser})
}
