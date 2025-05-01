package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func GetUsers(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var users []models.User
	db.DB.Find(&users)
	c.JSON(http.StatusOK, users)
}

func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" && fmt.Sprint(currentUser.ID) != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, id).Error; err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var targetUser models.User
	if err := db.DB.First(&targetUser, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err := db.DB.Delete(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	currentUser := c.MustGet("user").(models.User)

	log.Printf("Updating user ID: %s by user ID: %d, role: %s", id, currentUser.ID, currentUser.Role)

	if currentUser.Role != "admin" && fmt.Sprint(currentUser.ID) != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, id).Error; err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Аватарка
	file, err := c.FormFile("avatar")
	if err == nil {
		filename := fmt.Sprintf("Uploads/%s", file.Filename)
		log.Printf("Saving avatar to: %s", filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			log.Printf("Failed to save avatar: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
			return
		}
		targetUser.Avatar = filename
		log.Printf("Updated avatar path: %s", targetUser.Avatar)
	}

	// Дополнительные поля
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

	log.Printf("User updated successfully, avatar: %s", targetUser.Avatar)
	c.JSON(http.StatusOK, targetUser)
}

func BanUser(c *gin.Context) {
	id := c.Param("id")
	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	targetUser.IsBanned = true
	if err := db.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User banned", "user": targetUser})
}

func UnbanUser(c *gin.Context) {
	id := c.Param("id")
	currentUser := c.MustGet("user").(models.User)

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	targetUser.IsBanned = false
	if err := db.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unban user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unbanned", "user": targetUser})
}
