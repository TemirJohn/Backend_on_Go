package handlers

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// GetDashboardStats - simplified version without concurrent package
// This works immediately while you set up the concurrent package
func GetDashboardStats(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}

	start := time.Now()

	// Sequential stats calculation (will be replaced with concurrent later)
	var totalUsers, totalGames, totalReviews, totalSales int64
	var activeUsers, recentGames int64
	var avgRating float64

	// Count users
	db.DB.Model(&models.User{}).Count(&totalUsers)

	// Count games
	db.DB.Model(&models.Game{}).Count(&totalGames)

	// Count reviews
	db.DB.Model(&models.Review{}).Count(&totalReviews)

	// Count sales
	db.DB.Model(&models.Ownership{}).Count(&totalSales)

	// Count active users (not banned)
	db.DB.Model(&models.User{}).Where("is_banned = ?", false).Count(&activeUsers)

	// Count recent games (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	db.DB.Model(&models.Game{}).Where("created_at > ?", thirtyDaysAgo).Count(&recentGames)

	// Calculate average rating
	var avg struct{ Avg float64 }
	db.DB.Model(&models.Review{}).Select("AVG(rating) as avg").Scan(&avg)
	avgRating = avg.Avg

	// Get top category
	var topCategoryResult struct {
		CategoryID uint
		Count      int64
	}
	db.DB.Model(&models.Game{}).
		Select("category_id, COUNT(*) as count").
		Group("category_id").
		Order("count DESC").
		Limit(1).
		Scan(&topCategoryResult)

	topCategory := "N/A"
	if topCategoryResult.CategoryID != 0 {
		var category models.Category
		if err := db.DB.First(&category, topCategoryResult.CategoryID).Error; err == nil {
			topCategory = category.Name
		}
	}

	duration := time.Since(start)

	c.JSON(http.StatusOK, gin.H{
		"statistics": gin.H{
			"total_users":    totalUsers,
			"total_games":    totalGames,
			"total_reviews":  totalReviews,
			"total_sales":    totalSales,
			"active_users":   activeUsers,
			"recent_games":   recentGames,
			"average_rating": avgRating,
			"top_category":   topCategory,
		},
		"calculation_time": duration.String(),
	})
}
