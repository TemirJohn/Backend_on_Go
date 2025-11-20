package cash

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"time"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// InitRedis initializes Redis connection
func InitRedis() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     os.Getenv("REDIS_PASSWORD"), // пустой если нет пароля
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// CloseRedis closes Redis connection
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// IsRedisAvailable checks if Redis is connected
func IsRedisAvailable() bool {
	if RedisClient == nil {
		return false
	}
	_, err := RedisClient.Ping(ctx).Result()
	return err == nil
}

// ==================== CACHE KEYS ====================

const (
	// Game caching
	GameCachePrefix    = "game:"      // game:123
	GamesCacheKey      = "games:all"  // все игры
	GamesByCategoryKey = "games:cat:" // games:cat:5

	// User caching
	UserCachePrefix = "user:" // user:123

	// Category caching
	CategoryCacheKey = "categories:all" // все категории

	// Reviews caching
	ReviewsCachePrefix = "reviews:game:" // reviews:game:123

	// Library caching
	LibraryCachePrefix = "library:user:" // library:user:123

	// Statistics caching
	StatsCacheKey = "stats:dashboard" // статистика

	// Rate limiting
	RateLimitPrefix = "ratelimit:user:" // ratelimit:user:123
)

// ==================== GENERIC CACHE OPERATIONS ====================

// Set stores any value in cache with TTL
func Set(key string, value interface{}, ttl time.Duration) error {
	if !IsRedisAvailable() {
		return fmt.Errorf("redis not available")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return RedisClient.Set(ctx, key, data, ttl).Err()
}

// Get retrieves value from cache
func Get(key string, dest interface{}) error {
	if !IsRedisAvailable() {
		return fmt.Errorf("redis not available")
	}

	val, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes key from cache
func Delete(key string) error {
	if !IsRedisAvailable() {
		return nil
	}
	return RedisClient.Del(ctx, key).Err()
}

// DeletePattern removes all keys matching pattern
func DeletePattern(pattern string) error {
	if !IsRedisAvailable() {
		return nil
	}

	iter := RedisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := RedisClient.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Exists checks if key exists
func Exists(key string) (bool, error) {
	if !IsRedisAvailable() {
		return false, nil
	}

	result, err := RedisClient.Exists(ctx, key).Result()
	return result > 0, err
}

// ==================== GAME CACHING ====================

// GetGame returns cached game
func GetGame(gameID uint) (interface{}, error) {
	key := fmt.Sprintf("%s%d", GameCachePrefix, gameID)
	var game interface{}
	err := Get(key, &game)
	return game, err
}

// SetGame caches a game for 1 hour
func SetGame(gameID uint, game interface{}) error {
	key := fmt.Sprintf("%s%d", GameCachePrefix, gameID)
	return Set(key, game, time.Hour)
}

// InvalidateGame removes game from cache
func InvalidateGame(gameID uint) error {
	key := fmt.Sprintf("%s%d", GameCachePrefix, gameID)
	return Delete(key)
}

// GetGames returns all cached games
func GetGames() (interface{}, error) {
	var games interface{}
	err := Get(GamesCacheKey, &games)
	return games, err
}

// SetGames caches all games for 5 minutes
func SetGames(games interface{}) error {
	return Set(GamesCacheKey, games, 5*time.Minute)
}

// InvalidateGamesList invalidates the games list cache
func InvalidateGamesList() error {
	// Удаляем общий список
	if err := Delete(GamesCacheKey); err != nil {
		return err
	}
	// Удаляем все кеши игр по категориям
	return DeletePattern(GamesByCategoryKey + "*")
}

// GetGamesByCategory returns cached games for a category
func GetGamesByCategory(categoryID uint) (interface{}, error) {
	key := fmt.Sprintf("%s%d", GamesByCategoryKey, categoryID)
	var games interface{}
	err := Get(key, &games)
	return games, err
}

// SetGamesByCategory caches games for a category
func SetGamesByCategory(categoryID uint, games interface{}) error {
	key := fmt.Sprintf("%s%d", GamesByCategoryKey, categoryID)
	return Set(key, games, 5*time.Minute)
}

// ==================== USER CACHING ====================

// GetUser returns cached user
func GetUser(userID uint) (interface{}, error) {
	key := fmt.Sprintf("%s%d", UserCachePrefix, userID)
	var user interface{}
	err := Get(key, &user)
	return user, err
}

// SetUser caches a user for 30 minutes
func SetUser(userID uint, user interface{}) error {
	key := fmt.Sprintf("%s%d", UserCachePrefix, userID)
	return Set(key, user, 30*time.Minute)
}

// InvalidateUser removes user from cache
func InvalidateUser(userID uint) error {
	key := fmt.Sprintf("%s%d", UserCachePrefix, userID)
	return Delete(key)
}

// ==================== CATEGORY CACHING ====================

// GetCategories returns cached categories
func GetCategories() (interface{}, error) {
	var categories interface{}
	err := Get(CategoryCacheKey, &categories)
	return categories, err
}

// SetCategories caches categories for 1 hour
func SetCategories(categories interface{}) error {
	return Set(CategoryCacheKey, categories, time.Hour)
}

// InvalidateCategories removes categories cache
func InvalidateCategories() error {
	return Delete(CategoryCacheKey)
}

// ==================== REVIEWS CACHING ====================

// GetReviews returns cached reviews for a game
func GetReviews(gameID uint) (interface{}, error) {
	key := fmt.Sprintf("%s%d", ReviewsCachePrefix, gameID)
	var reviews interface{}
	err := Get(key, &reviews)
	return reviews, err
}

// SetReviews caches reviews for 10 minutes
func SetReviews(gameID uint, reviews interface{}) error {
	key := fmt.Sprintf("%s%d", ReviewsCachePrefix, gameID)
	return Set(key, reviews, 10*time.Minute)
}

// InvalidateReviews removes reviews cache for a game
func InvalidateReviews(gameID uint) error {
	key := fmt.Sprintf("%s%d", ReviewsCachePrefix, gameID)
	return Delete(key)
}

// ==================== LIBRARY CACHING ====================

// GetUserLibrary returns cached user library
func GetUserLibrary(userID uint) (interface{}, error) {
	key := fmt.Sprintf("%s%d", LibraryCachePrefix, userID)
	var library interface{}
	err := Get(key, &library)
	return library, err
}

// SetUserLibrary caches user library for 5 minutes
func SetUserLibrary(userID uint, library interface{}) error {
	key := fmt.Sprintf("%s%d", LibraryCachePrefix, userID)
	return Set(key, library, 5*time.Minute)
}

// InvalidateUserLibrary removes user library from cache
func InvalidateUserLibrary(userID uint) error {
	key := fmt.Sprintf("%s%d", LibraryCachePrefix, userID)
	return Delete(key)
}

// ==================== STATISTICS CACHING ====================

// GetDashboardStats returns cached dashboard statistics
func GetDashboardStats() (interface{}, error) {
	var stats interface{}
	err := Get(StatsCacheKey, &stats)
	return stats, err
}

// SetDashboardStats caches dashboard statistics for 5 minutes
func SetDashboardStats(stats interface{}) error {
	return Set(StatsCacheKey, stats, 5*time.Minute)
}

// InvalidateDashboardStats removes dashboard statistics cache
func InvalidateDashboardStats() error {
	return Delete(StatsCacheKey)
}

// ==================== RATE LIMITING ====================

// CheckRateLimit implements token bucket rate limiting
func CheckRateLimit(userID uint, maxRequests int, window time.Duration) (bool, int, error) {
	if !IsRedisAvailable() {
		return true, maxRequests, nil // Allow if Redis unavailable
	}

	key := fmt.Sprintf("%s%d", RateLimitPrefix, userID)

	// Get current count
	count, err := RedisClient.Get(ctx, key).Int()
	if err == redis.Nil {
		// First request - initialize counter
		if err := RedisClient.Set(ctx, key, 1, window).Err(); err != nil {
			return false, 0, err
		}
		return true, maxRequests - 1, nil
	}
	if err != nil {
		return false, 0, err
	}

	// Check if limit exceeded
	if count >= maxRequests {
		ttl, _ := RedisClient.TTL(ctx, key).Result()
		return false, 0, fmt.Errorf("rate limit exceeded, retry after %v", ttl)
	}

	// Increment counter
	newCount, err := RedisClient.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	remaining := maxRequests - int(newCount)
	return true, remaining, nil
}

// ResetRateLimit resets rate limit for a user
func ResetRateLimit(userID uint) error {
	key := fmt.Sprintf("%s%d", RateLimitPrefix, userID)
	return Delete(key)
}

// ==================== CACHE WARMING ====================

// WarmCache preloads commonly accessed data
func WarmCache() error {
	if !IsRedisAvailable() {
		return fmt.Errorf("redis not available")
	}

	// This function should be called on application startup
	// to preload frequently accessed data

	// Example: Preload categories
	// var categories []models.Category
	// db.DB.Find(&categories)
	// SetCategories(categories)

	return nil
}

// ==================== CACHE STATISTICS ====================

// GetCacheStats returns Redis statistics
func GetCacheStats() (map[string]interface{}, error) {
	if !IsRedisAvailable() {
		return nil, fmt.Errorf("redis not available")
	}

	info, err := RedisClient.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	dbSize, err := RedisClient.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"info":    info,
		"db_size": dbSize,
	}, nil
}

// FlushAll clears all cache (use with caution!)
func FlushAll() error {
	if !IsRedisAvailable() {
		return fmt.Errorf("redis not available")
	}
	return RedisClient.FlushAll(ctx).Err()
}
