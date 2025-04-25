package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Register(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := db.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Create new user
	user := models.User{
		Name:     input.Username,
		Email:    input.Email,
		Password: input.Password, // Password is already hashed by frontend
		Role:     input.Role,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}
