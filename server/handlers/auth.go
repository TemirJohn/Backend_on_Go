package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	var DB *gorm.DB // Replace it with the actual DB initialization as needed
	if err := DB.Where("email = ? AND password = ?", input.Email, input.Password).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("User-ID") // Simplified; use JWT in production
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
