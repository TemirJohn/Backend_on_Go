package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"awesomeProject/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

func Register(c *gin.Context) {
	var input models.RegisterInput

	// Bind form data
	input.Username = c.PostForm("username")
	input.Email = c.PostForm("email")
	input.Password = c.PostForm("password")
	input.Role = c.PostForm("role")

	// ДОБАВЛЕНО: Валидация
	if err := utils.ValidateStruct(input); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// Проверка существующего email
	var existingUser models.User
	if err := db.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Хеширование пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Загрузка аватара
	file, err := c.FormFile("avatar")
	var avatarPath string
	if err == nil {
		avatarPath = "uploads/" + file.Filename
		if err := c.SaveUploadedFile(file, avatarPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
			return
		}
		log.Printf("Avatar saved to: %s", avatarPath)
	}

	// Создание пользователя
	user := models.User{
		Name:     input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     input.Role,
		Avatar:   avatarPath,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}
