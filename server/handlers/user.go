package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"fmt"
	"github.com/gin-gonic/gin"
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

	if currentUser.Role != "admin" && fmt.Sprint(currentUser.ID) != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var targetUser models.User
	if err := db.DB.First(&targetUser, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Аватарка
	file, err := c.FormFile("avatar")
	if err == nil {
		filename := fmt.Sprintf("uploads/avatars/%s", file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
			return
		}
		targetUser.Avatar = filename
	}

	// Дополнительные поля
	var input struct {
		Name *string `form:"name"` // если multipart/form-data
		Role *string `form:"role"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != nil {
		targetUser.Name = *input.Name
	}
	if input.Role != nil && currentUser.Role == "admin" { // только админ может менять роль
		targetUser.Role = *input.Role
	}

	if err := db.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, targetUser)
}
