package db

import (
	"awesomeProject/models"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Загружаем строку подключения из .env
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=1123 user=postgres dbname=gamehub password=11466795 sslmode=disable"
	}

	// Инициализация глобальной переменной DB
	var openErr error
	DB, openErr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if openErr != nil {
		log.Fatal("failed to connect to the database:", openErr)
	}

	// Миграции
	migrateErr := DB.AutoMigrate(&models.User{}, &models.Game{}, &models.Ownership{}, &models.Category{}, &models.Review{}, &models.Purchase{})
	if migrateErr != nil {
		log.Fatal("failed to migrate:", migrateErr)
	}

	log.Println("Database connected and migrated")
}
